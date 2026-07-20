package sftpsvc

import (
	"context"
	"fmt"
	"io"
	"path"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"wiShell/backend/internal/model"
)

type TextFileStream struct {
	File    TextFile
	Content io.ReadCloser
}

type textFileReadCloser struct {
	file   *sftp.File
	client *sftp.Client
	once   sync.Once
	err    error
}

func OpenTextFileStream(ctx context.Context, conn model.Connection, remotePath string, timeout time.Duration) (TextFileStream, error) {
	if err := ctx.Err(); err != nil {
		return TextFileStream{}, err
	}

	sftpClient, err := textSFTPClient(conn, timeout)
	if err != nil {
		return TextFileStream{}, err
	}
	stopCancelWatch := make(chan struct{})
	defer close(stopCancelWatch)
	go func() {
		select {
		case <-ctx.Done():
			_ = sftpClient.Close()
		case <-stopCancelWatch:
		}
	}()

	closeClient := true
	defer func() {
		if closeClient {
			_ = sftpClient.Close()
		}
	}()

	resolved, err := resolveRemotePath(sftpClient, remotePath)
	if err != nil {
		return TextFileStream{}, fmt.Errorf("resolve file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return TextFileStream{}, err
	}

	stat, err := sftpClient.Stat(resolved)
	if err != nil {
		return TextFileStream{}, fmt.Errorf("stat file: %w", err)
	}
	if stat.IsDir() {
		return TextFileStream{}, fmt.Errorf("remote path is a directory")
	}
	if stat.Size() > MaxTextEditBytes {
		return TextFileStream{}, fmt.Errorf("remote file is too large for text editing: %d bytes", stat.Size())
	}

	file, err := sftpClient.Open(resolved)
	if err != nil {
		return TextFileStream{}, fmt.Errorf("open remote file: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		return TextFileStream{}, err
	}

	closeClient = false
	return TextFileStream{
		File: TextFile{
			Name:    path.Base(resolved),
			Path:    resolved,
			Size:    stat.Size(),
			ModTime: stat.ModTime().UTC().Format(time.RFC3339),
		},
		Content: &textFileReadCloser{file: file, client: sftpClient},
	}, nil
}

func (r *textFileReadCloser) Read(buffer []byte) (int, error) {
	return r.file.Read(buffer)
}

func (r *textFileReadCloser) Close() error {
	r.once.Do(func() {
		clientErr := r.client.Close()
		fileErr := r.file.Close()
		if clientErr != nil {
			r.err = clientErr
		} else {
			r.err = fileErr
		}
	})
	return r.err
}
