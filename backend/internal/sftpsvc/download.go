package sftpsvc

import (
	"fmt"
	"io"
	"path"
	"time"

	"github.com/pkg/sftp"
	"wiShell/backend/internal/model"
	"wiShell/backend/internal/sshsvc"
)

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
