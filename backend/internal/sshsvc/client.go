package sshsvc

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
	"zshell/backend/internal/model"
)

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func NewClient(conn model.Connection, timeout time.Duration) (*ssh.Client, error) {
	return dial(conn, timeout)
}

func TestConnection(conn model.Connection, timeout time.Duration) error {
	client, err := dial(conn, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	return nil
}

func ExecCommand(conn model.Connection, command string, timeout time.Duration) (ExecResult, error) {
	client, err := dial(conn, timeout)
	if err != nil {
		return ExecResult{}, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return ExecResult{}, fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	_ = session.Setenv("LANG", "C.UTF-8")
	_ = session.Setenv("LC_ALL", "C.UTF-8")

	runErr := session.Run(command)
	exitCode := 0

	if runErr != nil {
		if exitErr, ok := runErr.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		} else {
			return ExecResult{}, fmt.Errorf("run command: %w", runErr)
		}
	}

	return ExecResult{
		Stdout:   string(bytes.ToValidUTF8(stdoutBuf.Bytes(), []byte("?"))),
		Stderr:   string(bytes.ToValidUTF8(stderrBuf.Bytes(), []byte("?"))),
		ExitCode: exitCode,
	}, nil
}

func dial(conn model.Connection, timeout time.Duration) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            conn.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(conn.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	client, err := ssh.Dial("tcp", conn.Address(), config)
	if err != nil {
		return nil, fmt.Errorf("dial ssh: %w", err)
	}

	return client, nil
}
