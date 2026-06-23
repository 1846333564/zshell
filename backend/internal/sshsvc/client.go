package sshsvc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	auth, err := authMethods(conn)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            conn.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	client, err := ssh.Dial("tcp", conn.Address(), config)
	if err != nil {
		return nil, fmt.Errorf("dial ssh: %w", err)
	}

	return client, nil
}

func authMethods(conn model.Connection) ([]ssh.AuthMethod, error) {
	switch conn.AuthMethod {
	case "", "password":
		return []ssh.AuthMethod{ssh.Password(conn.Password)}, nil
	case "id_rsa":
		signer, err := loadDefaultIDRSA()
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", conn.AuthMethod)
	}
}

func loadDefaultIDRSA() (ssh.Signer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home: %w", err)
	}

	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", keyPath, err)
	}

	return signer, nil
}
