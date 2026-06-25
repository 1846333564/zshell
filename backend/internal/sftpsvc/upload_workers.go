package sftpsvc

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/pkg/sftp"
)

func uploadPreparedFiles(client *sftp.Client, files []preparedUploadFile, progress *uploadBatchProgress) ([]UploadResult, int64, error) {
	if len(files) == 0 {
		return nil, 0, nil
	}

	results := make([]UploadResult, len(files))
	jobs := make(chan preparedUploadFile)
	workerCount := uploadWorkerCount(len(files))
	var totalSize atomic.Int64
	var failed atomic.Bool
	var firstErr error
	var errOnce sync.Once
	var wg sync.WaitGroup

	fail := func(err error) {
		if err == nil {
			return
		}
		errOnce.Do(func() {
			firstErr = err
			failed.Store(true)
		})
	}

	wg.Add(workerCount)
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			defer wg.Done()
			for file := range jobs {
				if failed.Load() {
					continue
				}
				written, err := uploadPreparedFile(client, file, progress)
				if err != nil {
					fail(err)
					continue
				}
				results[file.index] = UploadResult{RemotePath: file.remotePath, Size: written}
				totalSize.Add(written)
				completed := progress.completeFile()
				progress.report(UploadProgressEvent{
					Stage:          "file",
					FileIndex:      file.index,
					FileName:       file.item.FileName,
					RelativePath:   file.relativePath,
					RemotePath:     file.remotePath,
					FileLoaded:     written,
					FileTotal:      file.item.Size,
					LoadedBytes:    progress.loaded(),
					TotalBytes:     progress.totalBytes,
					CompletedFiles: completed,
					TotalFiles:     progress.totalFiles,
					DirectoryCount: progress.directoryCount,
					Message:        "正在上传远程文件",
				})
			}
		}()
	}

	for _, file := range files {
		if failed.Load() {
			break
		}
		jobs <- file
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return nil, totalSize.Load(), firstErr
	}

	ordered := make([]UploadResult, 0, len(results))
	for _, result := range results {
		if result.RemotePath != "" {
			ordered = append(ordered, result)
		}
	}
	return ordered, totalSize.Load(), nil
}

func uploadPreparedFile(client *sftp.Client, file preparedUploadFile, progress *uploadBatchProgress) (int64, error) {
	progress.report(UploadProgressEvent{
		Stage:          "file",
		FileIndex:      file.index,
		FileName:       file.item.FileName,
		RelativePath:   file.relativePath,
		RemotePath:     file.remotePath,
		FileTotal:      file.item.Size,
		LoadedBytes:    progress.loaded(),
		TotalBytes:     progress.totalBytes,
		CompletedFiles: progress.completed(),
		TotalFiles:     progress.totalFiles,
		DirectoryCount: progress.directoryCount,
		Message:        "正在上传远程文件",
	})

	source, err := file.item.Open()
	if err != nil {
		return 0, fmt.Errorf("open upload source %s: %w", file.item.FileName, err)
	}

	var reported int64
	throttledReport := newUploadProgressThrottle(func(fileLoaded int64, force bool) {
		if fileLoaded > reported {
			progress.addLoaded(fileLoaded - reported)
			reported = fileLoaded
		}
		progress.report(UploadProgressEvent{
			Stage:          "file",
			FileIndex:      file.index,
			FileName:       file.item.FileName,
			RelativePath:   file.relativePath,
			RemotePath:     file.remotePath,
			FileLoaded:     fileLoaded,
			FileTotal:      file.item.Size,
			LoadedBytes:    progress.loaded(),
			TotalBytes:     progress.totalBytes,
			CompletedFiles: progress.completed(),
			TotalFiles:     progress.totalFiles,
			DirectoryCount: progress.directoryCount,
			Message:        "正在上传远程文件",
		})
	})

	written, uploadErr := uploadToPathWithProgress(client, file.remotePath, source, throttledReport)
	closeErr := source.Close()
	if uploadErr != nil {
		return 0, uploadErr
	}
	if closeErr != nil {
		return 0, fmt.Errorf("close upload source %s: %w", file.item.FileName, closeErr)
	}

	throttledReport(written, true)
	return written, nil
}

func uploadWorkerCount(fileCount int) int {
	if fileCount < uploadFileWorkerLimit {
		return fileCount
	}
	return uploadFileWorkerLimit
}
