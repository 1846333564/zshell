package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"wiShell/backend/internal/logsvc"
	"wiShell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPFileRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req sftpFileReadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ConnectionID) == "" || strings.TrimSpace(req.Path) == "" {
		writeError(w, http.StatusBadRequest, "connectionId and path are required")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	file, err := sftpsvc.ReadTextFile(conn, req.Path, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}

func (s *Server) handleSFTPFileReadRaw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req sftpFileReadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.ConnectionID = strings.TrimSpace(req.ConnectionID)
	req.Path = strings.TrimSpace(req.Path)
	if req.ConnectionID == "" || req.Path == "" {
		writeError(w, http.StatusBadRequest, "connectionId and path are required")
		return
	}
	if len(req.ConnectionID) > 256 || len(req.Path) > 8192 {
		writeError(w, http.StatusBadRequest, "remote file read field is too long")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	stream, err := sftpsvc.OpenTextFileStream(r.Context(), conn, req.Path, s.sshTimeout)
	if err != nil {
		if r.Context().Err() == nil {
			writeError(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	defer stream.Content.Close()

	stopCancelWatch := make(chan struct{})
	defer close(stopCancelWatch)
	go func() {
		select {
		case <-r.Context().Done():
			_ = stream.Content.Close()
		case <-stopCancelWatch:
		}
	}()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-WiShell-File-Size", strconv.FormatInt(stream.File.Size, 10))
	w.Header().Set("X-WiShell-File-Path", encodeTextResponseHeader(stream.File.Path))
	w.Header().Set("X-WiShell-File-Name", encodeTextResponseHeader(stream.File.Name))
	w.Header().Set("X-WiShell-File-Mod-Time", encodeTextResponseHeader(stream.File.ModTime))
	w.Header().Set("Content-Length", strconv.FormatInt(stream.File.Size, 10))
	w.WriteHeader(http.StatusOK)
	responseController := http.NewResponseController(w)
	if err := responseController.Flush(); err != nil {
		if r.Context().Err() == nil {
			logsvc.Error("SFTP 远程文本响应头刷新失败", err)
		}
		return
	}

	buffer := make([]byte, 64*1024)
	limitedReader := io.LimitReader(stream.Content, stream.File.Size)
	var writtenBytes int64
	for {
		readBytes, readErr := limitedReader.Read(buffer)
		if readBytes > 0 {
			written, writeErr := w.Write(buffer[:readBytes])
			writtenBytes += int64(written)
			if writeErr == nil && written != readBytes {
				writeErr = io.ErrShortWrite
			}
			if writeErr != nil {
				if r.Context().Err() == nil {
					logsvc.Error("SFTP 远程文本原始响应写入失败", writeErr)
				}
				return
			}
			if flushErr := responseController.Flush(); flushErr != nil {
				if r.Context().Err() == nil {
					logsvc.Error("SFTP 远程文本响应正文刷新失败", flushErr)
				}
				return
			}
		}
		if readErr == nil {
			continue
		}
		if errors.Is(readErr, io.EOF) {
			if writtenBytes != stream.File.Size {
				if r.Context().Err() == nil {
					logsvc.Error("SFTP 远程文本原始读取长度不足", io.ErrUnexpectedEOF)
				}
				return
			}
			return
		}
		if r.Context().Err() == nil {
			logsvc.Error("SFTP 远程文本原始读取失败", readErr)
		}
		return
	}
}

func encodeTextResponseHeader(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func (s *Server) handleSFTPFileReadStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpFileReadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ConnectionID) == "" || strings.TrimSpace(req.Path) == "" {
		writeError(w, http.StatusBadRequest, "connectionId and path are required")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	writeEvent := func(payload map[string]any) {
		_ = encoder.Encode(payload)
		flusher.Flush()
	}

	file, err := sftpsvc.StreamTextFileWithChunks(conn, req.Path, s.sshTimeout, func(event sftpsvc.TextReadProgressEvent) {
		writeEvent(map[string]any{
			"type":     "progress",
			"progress": event,
		})
	}, func(event sftpsvc.TextReadChunkEvent) {
		writeEvent(map[string]any{
			"type":  "chunk",
			"chunk": event,
		})
	})
	if err != nil {
		logsvc.Error("SFTP 远程文本流式读取失败", err)
		writeEvent(map[string]any{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	writeEvent(map[string]any{
		"type": "result",
		"file": file,
	})
}

func (s *Server) handleSFTPFileWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpFileWriteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ConnectionID) == "" || strings.TrimSpace(req.Path) == "" {
		writeError(w, http.StatusBadRequest, "connectionId and path are required")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	file, err := sftpsvc.WriteTextFile(conn, req.Path, req.Content, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}
