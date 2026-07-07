package sftpsvc

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"wiShell/backend/internal/model"
	"wiShell/backend/internal/sshsvc"
)

func UploadFiles(conn model.Connection, remoteDir string, files []UploadItem, directories []string, timeout time.Duration) (UploadBatchResult, error) {
	return UploadFilesWithProgress(conn, remoteDir, files, directories, timeout, nil)
}

func UploadFilesWithProgress(conn model.Connection, remoteDir string, files []UploadItem, directories []string, timeout time.Duration, report UploadProgressReporter) (UploadBatchResult, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return UploadBatchResult{}, err
	}

	sftpClient, err := newUploadSFTPClient(client)
	if err != nil {
		sshsvc.DropSharedClient(conn)
		return UploadBatchResult{}, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	resolvedDir, err := resolveRemotePath(sftpClient, remoteDir)
	if err != nil {
		return UploadBatchResult{}, fmt.Errorf("resolve dir: %w", err)
	}

	return runUploadBatch(client, sftpClient, resolvedDir, files, directories, report)
}

func UploadFile(conn model.Connection, remoteDir string, fileName string, source io.Reader, timeout time.Duration) (string, int64, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return "", 0, err
	}

	sftpClient, err := newUploadSFTPClient(client)
	if err != nil {
		sshsvc.DropSharedClient(conn)
		return "", 0, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	resolvedDir, err := resolveRemotePath(sftpClient, remoteDir)
	if err != nil {
		return "", 0, fmt.Errorf("resolve dir: %w", err)
	}

	targetPath := path.Join(resolvedDir, path.Base(fileName))
	written, err := uploadToPath(sftpClient, targetPath, source)
	if err != nil {
		return "", 0, err
	}

	return targetPath, written, nil
}

func cleanRelativePath(value string, fallbackName string) (string, error) {
	if value == "" {
		value = fallbackName
	}

	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimPrefix(value, "/")
	parts := strings.Split(value, "/")
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", fmt.Errorf("path traversal is not allowed")
		}
		cleanParts = append(cleanParts, part)
	}

	if len(cleanParts) == 0 {
		return "", nil
	}

	return path.Join(cleanParts...), nil
}

func uploadToPath(client *sftp.Client, remotePath string, source io.Reader) (int64, error) {
	return uploadToPathWithProgress(client, remotePath, source, nil)
}

func uploadToPathWithProgress(client *sftp.Client, remotePath string, source io.Reader, progress func(int64, bool)) (int64, error) {
	dst, err := client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return 0, fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer dst.Close()

	writer := io.Writer(dst)
	if progress != nil {
		writer = &uploadProgressWriter{
			writer:   dst,
			progress: progress,
		}
	}
	written, err := io.Copy(writer, source)
	if err != nil {
		return 0, fmt.Errorf("upload copy to %s: %w", remotePath, err)
	}

	return written, nil
}

type uploadProgressWriter struct {
	writer   io.Writer
	written  int64
	progress func(int64, bool)
}

func (w *uploadProgressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.written += int64(n)
		w.progress(w.written, false)
	}
	return n, err
}

func uploadTotalSize(files []UploadItem) int64 {
	total := int64(0)
	for _, item := range files {
		if item.Size > 0 {
			total += item.Size
		}
	}
	return total
}

func reportUploadProgress(report UploadProgressReporter, event UploadProgressEvent) {
	if report != nil {
		report(event)
	}
}

func newUploadProgressThrottle(report func(int64, bool)) func(int64, bool) {
	return newProgressThrottle(180*time.Millisecond, 512*1024, report)
}

func newProgressThrottle(minInterval time.Duration, minBytesDelta int64, report func(int64, bool)) func(int64, bool) {
	var last time.Time
	var lastBytes int64
	return func(fileLoaded int64, force bool) {
		if report == nil {
			return
		}
		now := time.Now()
		bytesDelta := fileLoaded - lastBytes
		shouldReport := force || last.IsZero() || now.Sub(last) >= minInterval || bytesDelta >= minBytesDelta
		if !shouldReport {
			return
		}
		last = now
		lastBytes = fileLoaded
		report(fileLoaded, force)
	}
}
