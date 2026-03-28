package sftpsvc

import (
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"zshell/backend/internal/model"
	"zshell/backend/internal/sshsvc"
)

type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
}

func ListDirectory(conn model.Connection, remotePath string, timeout time.Duration) (string, []Entry, error) {
	client, err := sshsvc.NewClient(conn, timeout)
	if err != nil {
		return "", nil, err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", nil, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	resolved, err := resolveRemotePath(sftpClient, remotePath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve path: %w", err)
	}

	items, err := sftpClient.ReadDir(resolved)
	if err != nil {
		return "", nil, fmt.Errorf("read dir: %w", err)
	}

	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		itemPath := path.Join(resolved, item.Name())
		entries = append(entries, Entry{
			Name:    item.Name(),
			Path:    itemPath,
			Size:    item.Size(),
			IsDir:   item.IsDir(),
			Mode:    item.Mode().String(),
			ModTime: item.ModTime().UTC().Format(time.RFC3339),
		})
	}

	return resolved, entries, nil
}

func UploadFile(conn model.Connection, remoteDir string, fileName string, source io.Reader, timeout time.Duration) (string, int64, error) {
	client, err := sshsvc.NewClient(conn, timeout)
	if err != nil {
		return "", 0, err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return "", 0, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	resolvedDir, err := resolveRemotePath(sftpClient, remoteDir)
	if err != nil {
		return "", 0, fmt.Errorf("resolve dir: %w", err)
	}

	targetPath := path.Join(resolvedDir, path.Base(fileName))
	dst, err := sftpClient.Create(targetPath)
	if err != nil {
		return "", 0, fmt.Errorf("create remote file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, source)
	if err != nil {
		return "", 0, fmt.Errorf("upload copy: %w", err)
	}

	return targetPath, written, nil
}

func DownloadFile(conn model.Connection, remotePath string, timeout time.Duration) (io.ReadCloser, string, int64, error) {
	client, err := sshsvc.NewClient(conn, timeout)
	if err != nil {
		return nil, "", 0, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, "", 0, fmt.Errorf("create sftp client: %w", err)
	}

	resolved, err := resolveRemotePath(sftpClient, remotePath)
	if err != nil {
		sftpClient.Close()
		client.Close()
		return nil, "", 0, fmt.Errorf("resolve file: %w", err)
	}

	stat, err := sftpClient.Stat(resolved)
	if err != nil {
		sftpClient.Close()
		client.Close()
		return nil, "", 0, fmt.Errorf("stat file: %w", err)
	}
	if stat.IsDir() {
		sftpClient.Close()
		client.Close()
		return nil, "", 0, fmt.Errorf("remote path is a directory")
	}

	file, err := sftpClient.Open(resolved)
	if err != nil {
		sftpClient.Close()
		client.Close()
		return nil, "", 0, fmt.Errorf("open remote file: %w", err)
	}

	return &downloadReadCloser{
		file:       file,
		sftpClient: sftpClient,
		sshClient:  client,
	}, path.Base(resolved), stat.Size(), nil
}

func normalizeRemotePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "~"
	}
	return trimmed
}

func resolveRemotePath(client *sftp.Client, input string) (string, error) {
	target := normalizeRemotePath(input)

	if target == "~" {
		// Most SFTP servers resolve "." to the user's home directory.
		return client.RealPath(".")
	}

	return client.RealPath(target)
}

type downloadReadCloser struct {
	file       io.ReadCloser
	sftpClient *sftp.Client
	sshClient  io.Closer
}

func (d *downloadReadCloser) Read(p []byte) (int, error) {
	return d.file.Read(p)
}

func (d *downloadReadCloser) Close() error {
	_ = d.file.Close()
	_ = d.sftpClient.Close()
	return d.sshClient.Close()
}
