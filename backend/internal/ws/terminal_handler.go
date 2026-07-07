package ws

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"wiShell/backend/internal/logsvc"
	"wiShell/backend/internal/sshsvc"
	"wiShell/backend/internal/store"

	"github.com/gorilla/websocket"
)

const (
	wsPongWait     = 75 * time.Second
	wsPingInterval = 25 * time.Second
	wsWriteWait    = 10 * time.Second
)

type TerminalHandler struct {
	store      *store.MemoryStore
	sshTimeout time.Duration
	upgrader   websocket.Upgrader
}

func NewTerminalHandler(connStore *store.MemoryStore, sshTimeout time.Duration) *TerminalHandler {
	return &TerminalHandler{
		store:      connStore,
		sshTimeout: sshTimeout,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *TerminalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	connectionID := strings.TrimSpace(r.URL.Query().Get("connectionId"))
	if connectionID == "" {
		logsvc.ErrorMessage("终端 WebSocket 建连失败", "missing connectionId")
		http.Error(w, "missing connectionId", http.StatusBadRequest)
		return
	}

	connCfg, ok := h.store.Get(connectionID)
	if !ok {
		logsvc.ErrorMessage("终端 WebSocket 建连失败", "connection not found")
		http.Error(w, "connection not found", http.StatusNotFound)
		return
	}

	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logsvc.Error("终端 WebSocket upgrade 失败", err)
		return
	}

	shell, err := sshsvc.NewInteractiveShell(connCfg, h.sshTimeout, 120, 32)
	if err != nil {
		logsvc.Error("终端 SSH 连接失败", err)
		_ = writeWSError(wsConn, "SSH_CONNECT_FAILED", err.Error())
		_ = wsConn.Close()
		return
	}

	session := &terminalSession{
		ws:     wsConn,
		shell:  shell,
		outCh:  make(chan Message, 256),
		doneCh: make(chan struct{}),
	}

	session.run()
}

type terminalSession struct {
	ws        *websocket.Conn
	shell     *sshsvc.InteractiveShell
	outCh     chan Message
	doneCh    chan struct{}
	closeOnce sync.Once
}

func (s *terminalSession) run() {
	defer logsvc.Recover("终端 WebSocket 会话")
	defer s.stop()

	s.goSafe("终端 WebSocket 写循环", s.writeLoop)
	s.goSafe("终端 SSH stdout 读取", func() { s.readSSHStream(s.shell.Stdout(), false) })
	s.goSafe("终端 SSH stderr 读取", func() { s.readSSHStream(s.shell.Stderr(), true) })
	s.goSafe("终端 SSH shell 退出监控", s.watchShellExit)

	s.readWSLoop()
}

func (s *terminalSession) goSafe(location string, fn func()) {
	go func() {
		defer logsvc.Recover(location)
		fn()
	}()
}

func (s *terminalSession) writeLoop() {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-s.outCh:
			_ = s.ws.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := s.ws.WriteJSON(msg); err != nil {
				s.stop()
				return
			}
		case <-ticker.C:
			_ = s.ws.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := s.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsWriteWait)); err != nil {
				s.stop()
				return
			}
		case <-s.doneCh:
			return
		}
	}
}

func (s *terminalSession) readWSLoop() {
	s.ws.SetReadLimit(8192)
	_ = s.ws.SetReadDeadline(time.Now().Add(wsPongWait))
	s.ws.SetPongHandler(func(string) error {
		return s.ws.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		_, raw, err := s.ws.ReadMessage()
		if err != nil {
			return
		}
		_ = s.ws.SetReadDeadline(time.Now().Add(wsPongWait))

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			s.emitError("BAD_REQUEST", "invalid message json")
			continue
		}

		switch msg.Type {
		case "input":
			var data InputData
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				s.emitError("BAD_REQUEST", "invalid input payload")
				continue
			}
			if err := s.shell.WriteInput(data.Text); err != nil {
				s.emitError("SSH_STDIN_FAILED", err.Error())
				return
			}
		case "resize":
			var data ResizeData
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				s.emitError("BAD_REQUEST", "invalid resize payload")
				continue
			}
			if err := s.shell.Resize(data.Cols, data.Rows); err != nil {
				s.emitError("SSH_RESIZE_FAILED", err.Error())
				continue
			}
			s.enqueue(Message{Type: "resize-ack"})
		case "ping":
			s.enqueue(Message{Type: "pong"})
		default:
			s.emitError("BAD_REQUEST", "unsupported message type")
		}
	}
}

func (s *terminalSession) readSSHStream(reader io.Reader, isStderr bool) {
	decoder := &utf8StreamDecoder{}
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			text := decoder.Decode(buf[:n])
			if text != "" {
				s.emitOutput(text, isStderr)
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				if tail := decoder.Flush(); tail != "" {
					s.emitOutput(tail, isStderr)
				}
				return
			}

			s.emitError("SSH_STREAM_FAILED", err.Error())
			s.stop()
			return
		}
	}
}

func (s *terminalSession) watchShellExit() {
	if err := s.shell.Wait(); err != nil && !errors.Is(err, io.EOF) {
		s.emitError("SSH_SHELL_EXIT", err.Error())
	}
	s.stop()
}

func (s *terminalSession) enqueue(msg Message) {
	select {
	case s.outCh <- msg:
	case <-s.doneCh:
	}
}

func (s *terminalSession) stop() {
	s.closeOnce.Do(func() {
		close(s.doneCh)
		_ = s.shell.Close()
		_ = s.ws.Close()
	})
}

func (s *terminalSession) emitOutput(text string, isStderr bool) {
	payload, _ := json.Marshal(OutputData{Text: text, Stderr: isStderr})
	s.enqueue(Message{Type: "output", Data: payload})
}

func (s *terminalSession) emitError(code string, message string) {
	logsvc.ErrorMessage("终端 WebSocket "+code, message)
	payload, _ := json.Marshal(ErrorData{Code: code, Message: message})
	s.enqueue(Message{Type: "error", Data: payload})
}

func writeWSError(wsConn *websocket.Conn, code string, message string) error {
	payload, _ := json.Marshal(ErrorData{Code: code, Message: message})
	return wsConn.WriteJSON(Message{Type: "error", Data: payload})
}

type utf8StreamDecoder struct {
	pending []byte
}

func (d *utf8StreamDecoder) Decode(chunk []byte) string {
	if len(chunk) == 0 && len(d.pending) == 0 {
		return ""
	}

	merged := make([]byte, 0, len(d.pending)+len(chunk))
	merged = append(merged, d.pending...)
	merged = append(merged, chunk...)

	valid, tail := splitValidUTF8(merged)
	d.pending = d.pending[:0]
	d.pending = append(d.pending, tail...)

	if len(valid) == 0 {
		return ""
	}

	return string(valid)
}

func (d *utf8StreamDecoder) Flush() string {
	if len(d.pending) == 0 {
		return ""
	}

	valid := bytes.ToValidUTF8(d.pending, []byte("?"))
	d.pending = nil
	return string(valid)
}

func splitValidUTF8(data []byte) ([]byte, []byte) {
	if len(data) == 0 {
		return nil, nil
	}

	if utf8.Valid(data) {
		return data, nil
	}

	start := len(data) - utf8.UTFMax
	if start < 0 {
		start = 0
	}

	for i := len(data); i >= start; i-- {
		if utf8.Valid(data[:i]) {
			valid := append([]byte(nil), data[:i]...)
			tail := append([]byte(nil), data[i:]...)
			return valid, tail
		}
	}

	repaired := bytes.ToValidUTF8(data, []byte("?"))
	return repaired, nil
}
