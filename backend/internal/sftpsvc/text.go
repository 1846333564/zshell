package sftpsvc

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"zshell/backend/internal/model"
	"zshell/backend/internal/sshsvc"
)

func ReadTextFile(conn model.Connection, remotePath string, timeout time.Duration) (TextFile, error) {
	return ReadTextFileWithProgress(conn, remotePath, timeout, nil)
}

func ReadTextFileWithProgress(conn model.Connection, remotePath string, timeout time.Duration, report TextReadProgressReporter) (TextFile, error) {
	reportTextReadProgress(report, TextReadProgressEvent{
		Stage:   "preparing",
		Path:    remotePath,
		Message: "正在准备读取远程文件",
	})

	sftpClient, err := textSFTPClient(conn, timeout)
	if err != nil {
		return TextFile{}, err
	}
	defer sftpClient.Close()

	resolved, err := resolveRemotePath(sftpClient, remotePath)
	if err != nil {
		return TextFile{}, fmt.Errorf("resolve file: %w", err)
	}
	reportTextReadProgress(report, TextReadProgressEvent{
		Stage:   "stat",
		Path:    resolved,
		Message: "正在读取远程文件信息",
	})

	stat, err := sftpClient.Stat(resolved)
	if err != nil {
		return TextFile{}, fmt.Errorf("stat file: %w", err)
	}
	if stat.IsDir() {
		return TextFile{}, fmt.Errorf("remote path is a directory")
	}
	if stat.Size() > MaxTextEditBytes {
		return TextFile{}, fmt.Errorf("remote file is too large for text editing: %d bytes", stat.Size())
	}
	reportTextReadProgress(report, TextReadProgressEvent{
		Stage:      "downloading",
		Path:       resolved,
		FileName:   path.Base(resolved),
		TotalBytes: stat.Size(),
		Message:    "正在下载远程文件内容",
	})

	file, err := sftpClient.Open(resolved)
	if err != nil {
		return TextFile{}, fmt.Errorf("open remote file: %w", err)
	}
	defer file.Close()

	data, err := readTextFileContentWithProgress(file, stat.Size(), report)
	if err != nil {
		return TextFile{}, fmt.Errorf("read remote file: %w", err)
	}
	if len(data) > MaxTextEditBytes {
		return TextFile{}, fmt.Errorf("remote file is too large for text editing")
	}
	reportTextReadProgress(report, TextReadProgressEvent{
		Stage:       "done",
		Path:        resolved,
		FileName:    path.Base(resolved),
		LoadedBytes: int64(len(data)),
		TotalBytes:  stat.Size(),
		Message:     "远程文件下载完成",
	})

	return TextFile{
		Name:    path.Base(resolved),
		Path:    resolved,
		Size:    stat.Size(),
		Content: string(data),
		ModTime: stat.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

func readTextFileContentWithProgress(source io.Reader, totalBytes int64, report TextReadProgressReporter) ([]byte, error) {
	var buffer bytes.Buffer
	if totalBytes > 0 && totalBytes <= MaxTextEditBytes {
		buffer.Grow(int(totalBytes))
	}

	loadedBytes := int64(0)
	progress := newProgressThrottle(80*time.Millisecond, 512*1024, func(fileLoaded int64, force bool) {
		reportTextReadProgress(report, TextReadProgressEvent{
			Stage:       "downloading",
			LoadedBytes: fileLoaded,
			TotalBytes:  totalBytes,
			Message:     "正在下载远程文件内容",
		})
	})
	progress(0, true)

	chunk := make([]byte, 32*1024)
	for {
		n, readErr := source.Read(chunk)
		if n > 0 {
			loadedBytes += int64(n)
			if loadedBytes > MaxTextEditBytes {
				return nil, fmt.Errorf("remote file is too large for text editing")
			}
			if _, err := buffer.Write(chunk[:n]); err != nil {
				return nil, err
			}
			progress(loadedBytes, false)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	progress(loadedBytes, true)

	return buffer.Bytes(), nil
}

func reportTextReadProgress(report TextReadProgressReporter, event TextReadProgressEvent) {
	if report != nil {
		report(event)
	}
}

func WriteTextFile(conn model.Connection, remotePath string, content string, timeout time.Duration) (TextFile, error) {
	if len([]byte(content)) > MaxTextEditBytes {
		return TextFile{}, fmt.Errorf("edited content is too large: %d bytes", len([]byte(content)))
	}

	sftpClient, err := textSFTPClient(conn, timeout)
	if err != nil {
		return TextFile{}, err
	}
	defer sftpClient.Close()

	resolved, err := resolveRemotePath(sftpClient, remotePath)
	if err != nil {
		return TextFile{}, fmt.Errorf("resolve file: %w", err)
	}

	stat, err := sftpClient.Stat(resolved)
	if err != nil {
		return TextFile{}, fmt.Errorf("stat file: %w", err)
	}
	if stat.IsDir() {
		return TextFile{}, fmt.Errorf("remote path is a directory")
	}

	written, err := uploadToPath(sftpClient, resolved, strings.NewReader(content))
	if err != nil {
		return TextFile{}, err
	}

	stat, err = sftpClient.Stat(resolved)
	if err != nil {
		return TextFile{}, fmt.Errorf("stat saved file: %w", err)
	}

	return TextFile{
		Name:    path.Base(resolved),
		Path:    resolved,
		Size:    written,
		Content: content,
		ModTime: stat.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

func textSFTPClient(conn model.Connection, timeout time.Duration) (*sftp.Client, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err == nil {
		return sftpClient, nil
	}

	sshsvc.DropSharedClient(conn)
	client, retryErr := sshsvc.SharedClient(conn, timeout)
	if retryErr != nil {
		return nil, fmt.Errorf("create sftp client: %w; reconnect failed: %w", err, retryErr)
	}
	sftpClient, retryErr = sftp.NewClient(client)
	if retryErr != nil {
		sshsvc.DropSharedClient(conn)
		return nil, fmt.Errorf("create sftp client after reconnect: %w", retryErr)
	}
	return sftpClient, nil
}
