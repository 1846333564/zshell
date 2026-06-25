package sftpsvc

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	uploadFileWorkerLimit              = 8
	uploadMaxConcurrentRequestsPerFile = 16
	uploadMkdirCommandMaxLen           = 24000
	uploadMkdirCommandMaxArgs          = 120
)

type preparedUploadFile struct {
	index        int
	item         UploadItem
	relativePath string
	remotePath   string
}

type uploadBatchProgress struct {
	reporter       UploadProgressReporter
	totalBytes     int64
	totalFiles     int
	directoryCount int
	loadedBytes    atomic.Int64
	completedFiles atomic.Int64
	mu             sync.Mutex
}

func newUploadSFTPClient(client *ssh.Client) (*sftp.Client, error) {
	return sftp.NewClient(
		client,
		sftp.UseConcurrentWrites(true),
		sftp.MaxConcurrentRequestsPerFile(uploadMaxConcurrentRequestsPerFile),
	)
}

func runUploadBatch(sshClient *ssh.Client, sftpClient *sftp.Client, resolvedDir string, files []UploadItem, directories []string, report UploadProgressReporter) (UploadBatchResult, error) {
	progress := &uploadBatchProgress{
		reporter:       report,
		totalBytes:     uploadTotalSize(files),
		totalFiles:     len(files),
		directoryCount: len(directories),
	}
	progress.report(UploadProgressEvent{
		Stage:          "preparing",
		TotalBytes:     progress.totalBytes,
		TotalFiles:     progress.totalFiles,
		DirectoryCount: progress.directoryCount,
		Message:        "正在准备远程上传",
	})

	preparedFiles, explicitDirs, dirsToCreate, err := prepareUploadBatch(resolvedDir, files, directories)
	if err != nil {
		return UploadBatchResult{}, err
	}

	result := UploadBatchResult{
		OK:          true,
		Files:       make([]UploadResult, 0, len(preparedFiles)),
		Directories: explicitDirs,
	}

	if len(dirsToCreate) > 0 {
		progress.report(UploadProgressEvent{
			Stage:          "directories",
			LoadedBytes:    progress.loaded(),
			TotalBytes:     progress.totalBytes,
			TotalFiles:     progress.totalFiles,
			DirectoryCount: progress.directoryCount,
			Message:        "正在批量创建远程目录",
		})
		if err := createRemoteUploadDirs(sshClient, sftpClient, dirsToCreate); err != nil {
			return UploadBatchResult{}, err
		}
	}

	uploadedFiles, totalSize, err := uploadPreparedFiles(sftpClient, preparedFiles, progress)
	if err != nil {
		return UploadBatchResult{}, err
	}
	result.Files = uploadedFiles
	result.TotalSize = totalSize

	progress.report(UploadProgressEvent{
		Stage:          "done",
		LoadedBytes:    result.TotalSize,
		TotalBytes:     progress.totalBytes,
		CompletedFiles: len(result.Files),
		TotalFiles:     progress.totalFiles,
		DirectoryCount: progress.directoryCount,
		Message:        "上传完成",
	})

	return result, nil
}

func prepareUploadBatch(resolvedDir string, files []UploadItem, directories []string) ([]preparedUploadFile, []string, []string, error) {
	preparedFiles := make([]preparedUploadFile, 0, len(files))
	explicitDirs := make([]string, 0, len(directories))
	dirSet := make(map[string]struct{}, len(directories)+len(files))

	for _, dir := range directories {
		relativeDir, err := cleanRelativePath(dir, "")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid directory path %q: %w", dir, err)
		}
		if relativeDir == "" {
			continue
		}

		remotePath := path.Join(resolvedDir, relativeDir)
		explicitDirs = append(explicitDirs, remotePath)
		dirSet[remotePath] = struct{}{}
	}

	for index, item := range files {
		targetRelative, err := cleanRelativePath(item.RelativePath, item.FileName)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid upload path %q: %w", item.RelativePath, err)
		}
		if targetRelative == "" {
			return nil, nil, nil, fmt.Errorf("invalid upload file name %q", item.FileName)
		}

		remotePath := path.Join(resolvedDir, targetRelative)
		if parent := path.Dir(remotePath); shouldCreateUploadDir(parent, resolvedDir) {
			dirSet[parent] = struct{}{}
		}
		preparedFiles = append(preparedFiles, preparedUploadFile{
			index:        index,
			item:         item,
			relativePath: targetRelative,
			remotePath:   remotePath,
		})
	}

	return preparedFiles, explicitDirs, sortedUploadDirs(dirSet), nil
}

func shouldCreateUploadDir(parent string, resolvedDir string) bool {
	return parent != "" && parent != "." && parent != "/" && parent != resolvedDir
}

func sortedUploadDirs(dirSet map[string]struct{}) []string {
	dirs := make([]string, 0, len(dirSet))
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool {
		leftDepth := strings.Count(dirs[i], "/")
		rightDepth := strings.Count(dirs[j], "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

func (p *uploadBatchProgress) addLoaded(delta int64) int64 {
	if delta <= 0 {
		return p.loaded()
	}
	return p.loadedBytes.Add(delta)
}

func (p *uploadBatchProgress) loaded() int64 {
	if p == nil {
		return 0
	}
	return p.loadedBytes.Load()
}

func (p *uploadBatchProgress) completeFile() int {
	if p == nil {
		return 0
	}
	return int(p.completedFiles.Add(1))
}

func (p *uploadBatchProgress) completed() int {
	if p == nil {
		return 0
	}
	return int(p.completedFiles.Load())
}

func (p *uploadBatchProgress) report(event UploadProgressEvent) {
	if p == nil || p.reporter == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	reportUploadProgress(p.reporter, event)
}
