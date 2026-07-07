package sshsvc

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"wiShell/backend/internal/model"
)

type InteractiveShell struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader

	doneCh    chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	closed    bool
}

func NewInteractiveShell(conn model.Connection, timeout time.Duration, cols int, rows int) (*InteractiveShell, error) {
	client, err := dial(conn, timeout)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("create session: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if cols <= 0 {
		cols = 120
	}
	if rows <= 0 {
		rows = 32
	}

	_ = session.Setenv("LANG", "C.UTF-8")
	_ = session.Setenv("LC_ALL", "C.UTF-8")
	_ = session.Setenv("TERM", "xterm-256color")

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("start shell: %w", err)
	}

	shell := &InteractiveShell{
		client:  client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		doneCh:  make(chan struct{}),
	}
	go shell.keepAliveLoop()

	return shell, nil
}

func (s *InteractiveShell) WriteInput(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("shell already closed")
	}

	_, err := io.WriteString(s.stdin, text)
	if err != nil {
		return fmt.Errorf("write stdin: %w", err)
	}

	return nil
}

func (s *InteractiveShell) Resize(cols int, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}

	if err := s.session.WindowChange(rows, cols); err != nil {
		return fmt.Errorf("window change: %w", err)
	}

	return nil
}

func (s *InteractiveShell) Stdout() io.Reader {
	return s.stdout
}

func (s *InteractiveShell) Stderr() io.Reader {
	return s.stderr
}

func (s *InteractiveShell) Wait() error {
	return s.session.Wait()
}

func (s *InteractiveShell) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		close(s.doneCh)
		_ = s.session.Close()
		err = s.client.Close()
	})
	return err
}

func (s *InteractiveShell) keepAliveLoop() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _, err := s.client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				_ = s.Close()
				return
			}
		case <-s.doneCh:
			return
		}
	}
}
