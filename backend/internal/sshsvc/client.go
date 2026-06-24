package sshsvc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"zshell/backend/internal/model"
)

type ExecResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

var sharedClients sync.Map

type sharedClient struct {
	client *ssh.Client
}

func NewClient(conn model.Connection, timeout time.Duration) (*ssh.Client, error) {
	return dial(conn, timeout)
}

func SharedClient(conn model.Connection, timeout time.Duration) (*ssh.Client, error) {
	key := sharedClientKey(conn)
	if cached, ok := sharedClients.Load(key); ok {
		client := cached.(*sharedClient).client
		if sshClientAlive(client) {
			return client, nil
		}
		_ = client.Close()
		sharedClients.Delete(key)
	}

	client, err := dial(conn, timeout)
	if err != nil {
		return nil, err
	}

	cached := &sharedClient{client: client}
	actual, loaded := sharedClients.LoadOrStore(key, cached)
	if loaded {
		_ = client.Close()
		existing := actual.(*sharedClient).client
		if sshClientAlive(existing) {
			return existing, nil
		}
		_ = existing.Close()
		sharedClients.Delete(key)
		return SharedClient(conn, timeout)
	}

	return client, nil
}

func DropSharedClient(conn model.Connection) {
	key := sharedClientKey(conn)
	if cached, ok := sharedClients.LoadAndDelete(key); ok {
		_ = cached.(*sharedClient).client.Close()
	}
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

func sshClientAlive(client *ssh.Client) bool {
	_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
	return err == nil
}

func sharedClientKey(conn model.Connection) string {
	passwordHash := sha256.Sum256([]byte(conn.Password))
	parts := []string{
		strings.TrimSpace(conn.ID),
		strings.TrimSpace(conn.Host),
		fmt.Sprintf("%d", conn.Port),
		strings.TrimSpace(conn.Username),
		strings.TrimSpace(conn.AuthMethod),
		hex.EncodeToString(passwordHash[:]),
	}
	return strings.Join(parts, "\x00")
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
