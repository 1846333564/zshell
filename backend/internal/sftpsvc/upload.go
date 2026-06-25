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

func UploadFiles(conn model.Connection, remoteDir string, files []UploadItem, directories []string, timeout time.Duration) (UploadBatchResult, error) {
	return UploadFilesWithProgress(conn, remoteDir, files, directories, timeout, nil)
}

func UploadFilesWithProgress(conn model.Connection, remoteDir string, files []UploadItem, directories []string, timeout time.Duration, report UploadProgressReporter) (UploadBatchResult, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return UploadBatchResult{}, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		sshsvc.DropSharedClient(conn)
		return UploadBatchResult{}, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	resolvedDir, err := resolveRemotePath(sftpClient, remoteDir)
	if err != nil {
		return UploadBatchResult{}, fmt.Errorf("resolve dir: %w", err)
	}

	result := UploadBatchResult{
		OK:          true,
		Files:       make([]UploadResult, 0, len(files)),
		Directories: make([]string, 0, len(directories)),
	}
	totalBytes := uploadTotalSize(files)
	loadedBytes := int64(0)
	completedFiles := 0
	reportUploadProgress(report, UploadProgressEvent{
		Stage:          "preparing",
		TotalBytes:     totalBytes,
		TotalFiles:     len(files),
		DirectoryCount: len(directories),
		Message:        "正在准备远程上传",
	})

	for _, dir := range directories {
		relativeDir, err := cleanRelativePath(dir, "")
		if err != nil {
			return UploadBatchResult{}, fmt.Errorf("invalid directory path %q: %w", dir, err)
		}
		if relativeDir == "" {
			continue
		}

		remotePath := path.Join(resolvedDir, relativeDir)
		if err := sftpClient.MkdirAll(remotePath); err != nil {
			return UploadBatchResult{}, fmt.Errorf("create remote directory %s: %w", remotePath, err)
		}
		result.Directories = append(result.Directories, remotePath)
	}

	for index, item := range files {
		targetRelative, err := cleanRelativePath(item.RelativePath, item.FileName)
		if err != nil {
			return UploadBatchResult{}, fmt.Errorf("invalid upload path %q: %w", item.RelativePath, err)
		}
		if targetRelative == "" {
			return UploadBatchResult{}, fmt.Errorf("invalid upload file name %q", item.FileName)
		}

		remotePath := path.Join(resolvedDir, targetRelative)
		reportUploadProgress(report, UploadProgressEvent{
			Stage:          "file",
			FileIndex:      index,
			FileName:       item.FileName,
			RelativePath:   targetRelative,
			RemotePath:     remotePath,
			FileTotal:      item.Size,
			LoadedBytes:    loadedBytes,
			TotalBytes:     totalBytes,
			CompletedFiles: completedFiles,
			TotalFiles:     len(files),
			DirectoryCount: len(directories),
			Message:        "正在上传远程文件",
		})
		if parent := path.Dir(remotePath); parent != "." && parent != "/" {
			if err := sftpClient.MkdirAll(parent); err != nil {
				return UploadBatchResult{}, fmt.Errorf("create remote parent %s: %w", parent, err)
			}
		}

		source, err := item.Open()
		if err != nil {
			return UploadBatchResult{}, fmt.Errorf("open upload source %s: %w", item.FileName, err)
		}

		throttledReport := newUploadProgressThrottle(func(fileLoaded int64, force bool) {
			reportUploadProgress(report, UploadProgressEvent{
				Stage:          "file",
				FileIndex:      index,
				FileName:       item.FileName,
				RelativePath:   targetRelative,
				RemotePath:     remotePath,
				FileLoaded:     fileLoaded,
				FileTotal:      item.Size,
				LoadedBytes:    loadedBytes + fileLoaded,
				TotalBytes:     totalBytes,
				CompletedFiles: completedFiles,
				TotalFiles:     len(files),
				DirectoryCount: len(directories),
				Message:        "正在上传远程文件",
			})
		})
		written, uploadErr := uploadToPathWithProgress(sftpClient, remotePath, source, throttledReport)
		closeErr := source.Close()
		if uploadErr != nil {
			return UploadBatchResult{}, uploadErr
		}
		if closeErr != nil {
			return UploadBatchResult{}, fmt.Errorf("close upload source %s: %w", item.FileName, closeErr)
		}

		throttledReport(written, true)
		loadedBytes += written
		completedFiles++
		result.Files = append(result.Files, UploadResult{RemotePath: remotePath, Size: written})
		result.TotalSize += written
	}
	reportUploadProgress(report, UploadProgressEvent{
		Stage:          "done",
		LoadedBytes:    result.TotalSize,
		TotalBytes:     totalBytes,
		CompletedFiles: completedFiles,
		TotalFiles:     len(files),
		DirectoryCount: len(directories),
		Message:        "上传完成",
	})

	return result, nil
}

func UploadFile(conn model.Connection, remoteDir string, fileName string, source io.Reader, timeout time.Duration) (string, int64, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return "", 0, err
	}

	sftpClient, err := sftp.NewClient(client)
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
	dst, err := client.Create(remotePath)
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
