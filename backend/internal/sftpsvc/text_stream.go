package sftpsvc

import (
	"fmt"
	"io"
	"path"
	"time"

	"zshell/backend/internal/model"
)

func StreamTextFileWithChunks(conn model.Connection, remotePath string, timeout time.Duration, report TextReadProgressReporter, reportChunk TextReadChunkReporter) (TextFile, error) {
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
	fileName := path.Base(resolved)
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
		FileName:   fileName,
		TotalBytes: stat.Size(),
		Message:    "正在下载远程文件内容",
	})

	file, err := sftpClient.Open(resolved)
	if err != nil {
		return TextFile{}, fmt.Errorf("open remote file: %w", err)
	}
	defer file.Close()

	if err := streamTextFileContent(file, resolved, fileName, stat.Size(), report, reportChunk); err != nil {
		return TextFile{}, fmt.Errorf("read remote file: %w", err)
	}
	reportTextReadProgress(report, TextReadProgressEvent{
		Stage:       "done",
		Path:        resolved,
		FileName:    fileName,
		LoadedBytes: stat.Size(),
		TotalBytes:  stat.Size(),
		Message:     "远程文件下载完成",
	})

	return TextFile{
		Name:    fileName,
		Path:    resolved,
		Size:    stat.Size(),
		ModTime: stat.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

func streamTextFileContent(source io.Reader, remotePath, fileName string, totalBytes int64, report TextReadProgressReporter, reportChunk TextReadChunkReporter) error {
	loadedBytes := int64(0)
	progress := newProgressThrottle(80*time.Millisecond, 512*1024, func(fileLoaded int64, force bool) {
		reportTextReadProgress(report, TextReadProgressEvent{
			Stage:       "downloading",
			Path:        remotePath,
			FileName:    fileName,
			LoadedBytes: fileLoaded,
			TotalBytes:  totalBytes,
			Message:     "正在下载远程文件内容",
		})
	})
	progress(0, true)

	chunk := make([]byte, TextStreamChunkBytes)
	for {
		n, readErr := source.Read(chunk)
		if n > 0 {
			offsetBytes := loadedBytes
			loadedBytes += int64(n)
			if loadedBytes > MaxTextEditBytes {
				return fmt.Errorf("remote file is too large for text editing")
			}
			if reportChunk != nil {
				data := make([]byte, n)
				copy(data, chunk[:n])
				reportChunk(TextReadChunkEvent{
					Path:        remotePath,
					FileName:    fileName,
					OffsetBytes: offsetBytes,
					LoadedBytes: loadedBytes,
					TotalBytes:  totalBytes,
					Data:        data,
				})
			}
			progress(loadedBytes, false)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	progress(loadedBytes, true)

	return nil
}
