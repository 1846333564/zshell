package sshsvc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

const hardwareInfoCommand = `threads="$(getconf _NPROCESSORS_ONLN 2>/dev/null || nproc 2>/dev/null || grep -c '^processor' /proc/cpuinfo 2>/dev/null || echo 1)"
cores="$(lscpu 2>/dev/null | awk -F: '/^Core\(s\) per socket/{gsub(/^[ \t]+|[ \t]+$/, "", $2); cores=$2} /^Socket\(s\)/{gsub(/^[ \t]+|[ \t]+$/, "", $2); sockets=$2} END{if (cores != "" && sockets != "") print cores * sockets}')"
model="$(awk -F: '/model name|Hardware|Processor/{gsub(/^[ \t]+|[ \t]+$/, "", $2); print $2; exit}' /proc/cpuinfo 2>/dev/null)"
memkb="$(awk '/MemTotal/{print $2; exit}' /proc/meminfo 2>/dev/null)"
printf 'cpu_threads=%s\n' "$threads"
printf 'cpu_cores=%s\n' "$cores"
printf 'cpu_model=%s\n' "$model"
printf 'memory_kb=%s\n' "$memkb"`

const (
	sharedClientProbeTimeout  = 800 * time.Millisecond
	sharedClientProbeInterval = 5 * time.Second
)

var sharedClients sync.Map

type sharedClient struct {
	client      *ssh.Client
	mu          sync.Mutex
	lastProbe   time.Time
	lastProbeOK bool
}

func NewClient(conn model.Connection, timeout time.Duration) (*ssh.Client, error) {
	return dial(conn, timeout)
}

func SharedClient(conn model.Connection, timeout time.Duration) (*ssh.Client, error) {
	key := sharedClientKey(conn)
	if cached, ok := sharedClients.Load(key); ok {
		cachedClient := cached.(*sharedClient)
		if cachedClient.alive() {
			return cachedClient.client, nil
		}
		_ = cachedClient.client.Close()
		sharedClients.Delete(key)
	}

	client, err := dial(conn, timeout)
	if err != nil {
		return nil, err
	}

	cached := &sharedClient{
		client:      client,
		lastProbe:   time.Now(),
		lastProbeOK: true,
	}
	actual, loaded := sharedClients.LoadOrStore(key, cached)
	if loaded {
		_ = client.Close()
		existing := actual.(*sharedClient)
		if existing.alive() {
			return existing.client, nil
		}
		_ = existing.client.Close()
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

func (c *sharedClient) alive() bool {
	c.mu.Lock()
	if !c.lastProbe.IsZero() && time.Since(c.lastProbe) < sharedClientProbeInterval {
		ok := c.lastProbeOK
		c.mu.Unlock()
		return ok
	}
	c.mu.Unlock()

	ok := sshClientAlive(c.client)

	c.mu.Lock()
	c.lastProbe = time.Now()
	c.lastProbeOK = ok
	c.mu.Unlock()

	return ok
}

func TestConnection(conn model.Connection, timeout time.Duration) error {
	client, err := dial(conn, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	return nil
}

func ReadHardwareInfo(conn model.Connection, timeout time.Duration) (model.HardwareInfo, error) {
	result, err := ExecCommand(conn, hardwareInfoCommand, timeout)
	if err != nil {
		return model.HardwareInfo{}, err
	}
	if result.ExitCode != 0 {
		return model.HardwareInfo{}, fmt.Errorf("read hardware info failed: %s", strings.TrimSpace(result.Stderr))
	}

	values := parseKeyValueLines(result.Stdout)
	hardware := model.HardwareInfo{
		CPUThreads:       parsePositiveInt(values["cpu_threads"]),
		CPUCores:         parsePositiveInt(values["cpu_cores"]),
		CPUModel:         strings.TrimSpace(values["cpu_model"]),
		MemoryTotalBytes: int64(parsePositiveInt(values["memory_kb"])) * 1024,
		ReadAt:           time.Now().UTC().Format(time.RFC3339),
	}
	if hardware.CPUThreads <= 0 {
		hardware.CPUThreads = 1
	}
	return hardware, nil
}

func ExecCommand(conn model.Connection, command string, timeout time.Duration) (ExecResult, error) {
	client, err := dial(conn, timeout)
	if err != nil {
		return ExecResult{}, err
	}
	defer client.Close()

	return runCommand(client, command)
}

func ExecCommandShared(conn model.Connection, command string, timeout time.Duration) (ExecResult, error) {
	client, err := SharedClient(conn, timeout)
	if err != nil {
		return ExecResult{}, err
	}

	result, err := runCommand(client, command)
	if err != nil {
		DropSharedClient(conn)
	}
	return result, err
}

func runCommand(client *ssh.Client, command string) (ExecResult, error) {
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

func parseKeyValueLines(value string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(value, "\n") {
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(raw)
	}
	return result
}

func parsePositiveInt(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 0 {
		return 0
	}
	return number
}

func sshClientAlive(client *ssh.Client) bool {
	done := make(chan error, 1)
	go func() {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		done <- err
	}()

	timer := time.NewTimer(sharedClientProbeTimeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return err == nil
	case <-timer.C:
		return false
	}
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
