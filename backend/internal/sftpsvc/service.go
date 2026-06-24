package sftpsvc

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"zshell/backend/internal/model"
	"zshell/backend/internal/sshsvc"
)

type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	Mode    string `json:"mode"`
	Owner   string `json:"owner"`
	ModTime string `json:"modTime"`
}

const MaxTextEditBytes = 10 * 1024 * 1024

type TextFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Content string `json:"content"`
	ModTime string `json:"modTime"`
}

type UploadItem struct {
	FileName     string
	RelativePath string
	Size         int64
	Open         func() (io.ReadCloser, error)
}

type UploadResult struct {
	RemotePath string `json:"remotePath"`
	Size       int64  `json:"size"`
}

type UploadBatchResult struct {
	OK          bool           `json:"ok"`
	Files       []UploadResult `json:"files"`
	Directories []string       `json:"directories"`
	TotalSize   int64          `json:"totalSize"`
}

type UploadProgressEvent struct {
	Stage          string `json:"stage"`
	FileIndex      int    `json:"fileIndex"`
	FileName       string `json:"fileName,omitempty"`
	RelativePath   string `json:"relativePath,omitempty"`
	RemotePath     string `json:"remotePath,omitempty"`
	FileLoaded     int64  `json:"fileLoaded"`
	FileTotal      int64  `json:"fileTotal"`
	LoadedBytes    int64  `json:"loadedBytes"`
	TotalBytes     int64  `json:"totalBytes"`
	CompletedFiles int    `json:"completedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	DirectoryCount int    `json:"directoryCount"`
	Message        string `json:"message,omitempty"`
}

type UploadProgressReporter func(UploadProgressEvent)

type TransferItem struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

type TransferResult struct {
	RemotePath string `json:"remotePath"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
}

type TransferBatchResult struct {
	OK          bool             `json:"ok"`
	Action      string           `json:"action"`
	Files       []TransferResult `json:"files"`
	Directories []string         `json:"directories"`
	TotalSize   int64            `json:"totalSize"`
}

type DeleteResult struct {
	RemotePath string `json:"remotePath"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
}

type DeleteBatchResult struct {
	OK        bool           `json:"ok"`
	Items     []DeleteResult `json:"items"`
	TotalSize int64          `json:"totalSize"`
}

func ListDirectory(conn model.Connection, remotePath string, timeout time.Duration) (string, []Entry, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return "", nil, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		sshsvc.DropSharedClient(conn)
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
			Owner:   ownerFromFileInfo(item),
			ModTime: item.ModTime().UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	return resolved, entries, nil
}

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

func ReadTextFile(conn model.Connection, remotePath string, timeout time.Duration) (TextFile, error) {
	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return TextFile{}, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		sshsvc.DropSharedClient(conn)
		return TextFile{}, fmt.Errorf("create sftp client: %w", err)
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
	if stat.Size() > MaxTextEditBytes {
		return TextFile{}, fmt.Errorf("remote file is too large for text editing: %d bytes", stat.Size())
	}

	file, err := sftpClient.Open(resolved)
	if err != nil {
		return TextFile{}, fmt.Errorf("open remote file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, MaxTextEditBytes+1))
	if err != nil {
		return TextFile{}, fmt.Errorf("read remote file: %w", err)
	}
	if len(data) > MaxTextEditBytes {
		return TextFile{}, fmt.Errorf("remote file is too large for text editing")
	}

	return TextFile{
		Name:    path.Base(resolved),
		Path:    resolved,
		Size:    stat.Size(),
		Content: string(data),
		ModTime: stat.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

func WriteTextFile(conn model.Connection, remotePath string, content string, timeout time.Duration) (TextFile, error) {
	if len([]byte(content)) > MaxTextEditBytes {
		return TextFile{}, fmt.Errorf("edited content is too large: %d bytes", len([]byte(content)))
	}

	client, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return TextFile{}, err
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		sshsvc.DropSharedClient(conn)
		return TextFile{}, fmt.Errorf("create sftp client: %w", err)
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

func ArchiveItems(conn model.Connection, remotePaths []string, target io.Writer, timeout time.Duration) error {
	client, err := sshsvc.NewClient(conn, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	archive := zip.NewWriter(target)
	defer archive.Close()

	for _, remotePath := range remotePaths {
		resolved, err := resolveRemotePath(sftpClient, remotePath)
		if err != nil {
			return fmt.Errorf("resolve archive item %s: %w", remotePath, err)
		}

		stat, err := sftpClient.Stat(resolved)
		if err != nil {
			return fmt.Errorf("stat archive item %s: %w", resolved, err)
		}

		baseName := path.Base(resolved)
		if stat.IsDir() {
			if err := addZipDir(sftpClient, archive, resolved, baseName, stat); err != nil {
				return err
			}
			continue
		}

		if err := addZipFile(sftpClient, archive, resolved, baseName, stat); err != nil {
			return err
		}
	}

	return nil
}

func TransferItems(sourceConn model.Connection, targetConn model.Connection, targetDir string, items []TransferItem, action string, timeout time.Duration) (TransferBatchResult, error) {
	if action != "copy" && action != "move" {
		return TransferBatchResult{}, fmt.Errorf("unsupported transfer action: %s", action)
	}
	if len(items) == 0 {
		return TransferBatchResult{}, fmt.Errorf("no transfer items")
	}

	sourceSSH, err := sshsvc.SharedClient(sourceConn, timeout)
	if err != nil {
		return TransferBatchResult{}, err
	}

	sourceSFTP, err := sftp.NewClient(sourceSSH)
	if err != nil {
		sshsvc.DropSharedClient(sourceConn)
		return TransferBatchResult{}, fmt.Errorf("create source sftp client: %w", err)
	}
	defer sourceSFTP.Close()

	sameConnection := isSameConnection(sourceConn, targetConn)
	if sameConnection {
		resolvedTargetDir, err := resolveRemotePath(sourceSFTP, targetDir)
		if err != nil {
			return TransferBatchResult{}, fmt.Errorf("resolve target dir: %w", err)
		}
		return transferItemsOnSameConnection(sourceSSH, sourceSFTP, resolvedTargetDir, items, action)
	}

	targetSSH, err := sshsvc.SharedClient(targetConn, timeout)
	if err != nil {
		return TransferBatchResult{}, err
	}

	targetSFTP, err := sftp.NewClient(targetSSH)
	if err != nil {
		sshsvc.DropSharedClient(targetConn)
		return TransferBatchResult{}, fmt.Errorf("create target sftp client: %w", err)
	}
	defer targetSFTP.Close()

	resolvedTargetDir, err := resolveRemotePath(targetSFTP, targetDir)
	if err != nil {
		return TransferBatchResult{}, fmt.Errorf("resolve target dir: %w", err)
	}

	result := TransferBatchResult{
		OK:          true,
		Action:      action,
		Files:       make([]TransferResult, 0, len(items)),
		Directories: make([]string, 0),
	}

	for _, item := range items {
		sourcePath, err := resolveRemotePath(sourceSFTP, item.Path)
		if err != nil {
			return TransferBatchResult{}, fmt.Errorf("resolve source path %s: %w", item.Path, err)
		}

		stat, err := sourceSFTP.Stat(sourcePath)
		if err != nil {
			return TransferBatchResult{}, fmt.Errorf("stat source path %s: %w", sourcePath, err)
		}

		targetPath := path.Join(resolvedTargetDir, path.Base(sourcePath))
		if sameConnection && stat.IsDir() && isSameOrChildPath(resolvedTargetDir, sourcePath) {
			return TransferBatchResult{}, fmt.Errorf("cannot %s directory %s into itself or a child directory", action, sourcePath)
		}
		if action == "copy" {
			targetPath, err = availableCopyTargetPath(targetSFTP, sourcePath, targetPath, sameConnection)
			if err != nil {
				return TransferBatchResult{}, err
			}
		}
		if action == "move" && sameConnection && targetPath == sourcePath {
			if stat.IsDir() {
				result.Directories = append(result.Directories, targetPath)
			} else {
				result.Files = append(result.Files, TransferResult{RemotePath: targetPath, Size: stat.Size()})
			}
			continue
		}

		if stat.IsDir() {
			files, dirs, bytesCopied, err := copyRemoteDir(sourceSFTP, targetSFTP, sourcePath, targetPath)
			if err != nil {
				return TransferBatchResult{}, err
			}
			result.Files = append(result.Files, files...)
			result.Directories = append(result.Directories, dirs...)
			result.TotalSize += bytesCopied
		} else {
			written, err := copyRemoteFile(sourceSFTP, targetSFTP, sourcePath, targetPath)
			if err != nil {
				return TransferBatchResult{}, err
			}
			result.Files = append(result.Files, TransferResult{RemotePath: targetPath, Size: written})
			result.TotalSize += written
		}

		if action == "move" {
			if err := removeRemote(sourceSFTP, sourcePath); err != nil {
				return TransferBatchResult{}, fmt.Errorf("remove source path %s after move: %w", sourcePath, err)
			}
		}
	}

	return result, nil
}

func transferItemsOnSameConnection(sshClient *ssh.Client, sftpClient *sftp.Client, resolvedTargetDir string, items []TransferItem, action string) (TransferBatchResult, error) {
	result := TransferBatchResult{
		OK:          true,
		Action:      action,
		Files:       make([]TransferResult, 0, len(items)),
		Directories: make([]string, 0),
	}

	for _, item := range items {
		sourcePath, err := resolveRemotePath(sftpClient, item.Path)
		if err != nil {
			return TransferBatchResult{}, fmt.Errorf("resolve source path %s: %w", item.Path, err)
		}

		stat, err := sftpClient.Stat(sourcePath)
		if err != nil {
			return TransferBatchResult{}, fmt.Errorf("stat source path %s: %w", sourcePath, err)
		}

		targetPath := path.Join(resolvedTargetDir, path.Base(sourcePath))
		if stat.IsDir() && isSameOrChildPath(resolvedTargetDir, sourcePath) {
			return TransferBatchResult{}, fmt.Errorf("cannot %s directory %s into itself or a child directory", action, sourcePath)
		}
		if action == "copy" {
			targetPath, err = availableCopyTargetPath(sftpClient, sourcePath, targetPath, true)
			if err != nil {
				return TransferBatchResult{}, err
			}
		}
		if action == "move" && targetPath == sourcePath {
			appendTransferResult(&result, targetPath, stat)
			continue
		}

		command := sameConnectionTransferCommand(action, sourcePath, targetPath)
		if err := runRemoteShellCommand(sshClient, command); err != nil {
			return TransferBatchResult{}, err
		}

		appendTransferResult(&result, targetPath, stat)
	}

	return result, nil
}

func appendTransferResult(result *TransferBatchResult, remotePath string, stat os.FileInfo) {
	if stat.IsDir() {
		result.Directories = append(result.Directories, remotePath)
		return
	}

	result.Files = append(result.Files, TransferResult{RemotePath: remotePath, Size: stat.Size()})
	result.TotalSize += stat.Size()
}

func DeleteItems(conn model.Connection, items []TransferItem, timeout time.Duration) (DeleteBatchResult, error) {
	if len(items) == 0 {
		return DeleteBatchResult{}, fmt.Errorf("no delete items")
	}

	deleteArgs := make([]string, 0, len(items))
	result := DeleteBatchResult{
		OK:    true,
		Items: make([]DeleteResult, 0, len(items)),
	}

	for _, item := range items {
		remotePath, shellArg, err := deleteShellPathArg(item.Path)
		if err != nil {
			return DeleteBatchResult{}, err
		}
		deleteArgs = append(deleteArgs, shellArg)
		result.Items = append(result.Items, DeleteResult{
			RemotePath: remotePath,
			IsDir:      item.IsDir,
			Size:       0,
		})
	}

	sshClient, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return DeleteBatchResult{}, err
	}

	if err := runRemoteShellCommand(sshClient, "rm -rf -- "+strings.Join(deleteArgs, " ")); err != nil {
		return DeleteBatchResult{}, err
	}

	return result, nil
}

func isSameConnection(left model.Connection, right model.Connection) bool {
	if left.ID != "" || right.ID != "" {
		return left.ID == right.ID
	}
	return left.Host == right.Host && left.Port == right.Port && left.Username == right.Username && left.AuthMethod == right.AuthMethod
}

func availableCopyTargetPath(client *sftp.Client, sourcePath string, targetPath string, sameConnection bool) (string, error) {
	candidate := targetPath
	for index := 1; index < 1000; index += 1 {
		if sameConnection && candidate == sourcePath {
			candidate = copyPathCandidate(targetPath, index)
			continue
		}

		exists, err := remotePathExists(client, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = copyPathCandidate(targetPath, index)
	}

	return "", fmt.Errorf("find available copy target for %s: too many conflicts", targetPath)
}

func remotePathExists(client *sftp.Client, remotePath string) (bool, error) {
	if _, err := client.Stat(remotePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat target path %s: %w", remotePath, err)
	}
	return true, nil
}

func copyPathCandidate(originalPath string, index int) string {
	dir := path.Dir(originalPath)
	base := path.Base(originalPath)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = base
		ext = ""
	}

	suffix := " copy"
	if index > 1 {
		suffix = fmt.Sprintf(" copy %d", index)
	}
	return path.Join(dir, name+suffix+ext)
}

func sameConnectionTransferCommand(action string, sourcePath string, targetPath string) string {
	sourceArg := shellQuote(sourcePath)
	targetArg := shellQuote(targetPath)
	if action == "copy" {
		return fmt.Sprintf("cp -a --reflink=auto -- %s %s 2>/dev/null || cp -a -- %s %s", sourceArg, targetArg, sourceArg, targetArg)
	}
	return fmt.Sprintf("mv -f -- %s %s", sourceArg, targetArg)
}

func deleteShellPathArg(remotePath string) (string, string, error) {
	requestedPath := normalizeRemotePath(remotePath)
	cleaned := path.Clean(requestedPath)
	if cleaned == "~" || isProtectedDeletePath(cleaned) {
		return "", "", fmt.Errorf("refuse to delete protected path: %s", requestedPath)
	}
	if strings.HasPrefix(cleaned, "~/") {
		suffix := strings.TrimPrefix(cleaned, "~")
		if suffix == "" || suffix == "/" {
			return "", "", fmt.Errorf("refuse to delete protected path: %s", requestedPath)
		}
		return cleaned, "${HOME}" + shellQuote(suffix), nil
	}
	if !strings.HasPrefix(cleaned, "/") {
		return "", "", fmt.Errorf("delete path must be absolute or under home: %s", requestedPath)
	}
	return cleaned, shellQuote(cleaned), nil
}

func runRemoteShellCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create remote command session: %w", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	_ = session.Setenv("LANG", "C.UTF-8")
	_ = session.Setenv("LC_ALL", "C.UTF-8")

	if err := session.Run(command); err != nil {
		output := strings.TrimSpace(string(bytes.ToValidUTF8(stderrBuf.Bytes(), []byte("?"))))
		if output == "" {
			output = strings.TrimSpace(string(bytes.ToValidUTF8(stdoutBuf.Bytes(), []byte("?"))))
		}
		if exitErr, ok := err.(*ssh.ExitError); ok {
			if output == "" {
				output = "no stderr"
			}
			return fmt.Errorf("remote command failed with exit %d: %s", exitErr.ExitStatus(), output)
		}
		return fmt.Errorf("run remote command: %w", err)
	}

	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func isSameOrChildPath(candidate string, parent string) bool {
	cleanCandidate := path.Clean(candidate)
	cleanParent := path.Clean(parent)
	if cleanCandidate == cleanParent {
		return true
	}
	if cleanParent == "/" {
		return strings.HasPrefix(cleanCandidate, "/")
	}
	return strings.HasPrefix(cleanCandidate, cleanParent+"/")
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

func ownerFromFileInfo(info os.FileInfo) string {
	stat, ok := info.Sys().(*sftp.FileStat)
	if !ok || stat == nil {
		return "-"
	}
	return fmt.Sprintf("%d:%d", stat.UID, stat.GID)
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
	var last time.Time
	return func(fileLoaded int64, force bool) {
		if report == nil {
			return
		}
		now := time.Now()
		if !force && !last.IsZero() && now.Sub(last) < 180*time.Millisecond {
			return
		}
		last = now
		report(fileLoaded, force)
	}
}

func addZipFile(client *sftp.Client, archive *zip.Writer, remotePath string, zipPath string, info os.FileInfo) error {
	zipPath = cleanZipPath(zipPath)
	if zipPath == "" {
		return fmt.Errorf("empty zip file path for %s", remotePath)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("create zip header for %s: %w", remotePath, err)
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip file %s: %w", zipPath, err)
	}

	source, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file %s: %w", remotePath, err)
	}
	defer source.Close()

	if _, err := io.Copy(writer, source); err != nil {
		return fmt.Errorf("write zip file %s: %w", zipPath, err)
	}

	return nil
}

func addZipDir(client *sftp.Client, archive *zip.Writer, remotePath string, zipPath string, info os.FileInfo) error {
	zipPath = cleanZipPath(zipPath)
	if zipPath == "" {
		return fmt.Errorf("empty zip directory path for %s", remotePath)
	}

	dirName := strings.TrimSuffix(zipPath, "/") + "/"
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("create zip directory header for %s: %w", remotePath, err)
	}
	header.Name = dirName
	header.Method = zip.Store

	if _, err := archive.CreateHeader(header); err != nil {
		return fmt.Errorf("create zip directory %s: %w", dirName, err)
	}

	items, err := client.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("read remote directory %s: %w", remotePath, err)
	}

	for _, item := range items {
		childRemotePath := path.Join(remotePath, item.Name())
		childZipPath := path.Join(strings.TrimSuffix(dirName, "/"), item.Name())
		if item.IsDir() {
			if err := addZipDir(client, archive, childRemotePath, childZipPath, item); err != nil {
				return err
			}
			continue
		}
		if err := addZipFile(client, archive, childRemotePath, childZipPath, item); err != nil {
			return err
		}
	}

	return nil
}

func cleanZipPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimPrefix(value, "/")
	parts := strings.Split(value, "/")
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			continue
		}
		cleanParts = append(cleanParts, part)
	}
	if len(cleanParts) == 0 {
		return ""
	}
	return path.Join(cleanParts...)
}

func copyRemoteFile(sourceClient *sftp.Client, targetClient *sftp.Client, sourcePath string, targetPath string) (int64, error) {
	if parent := path.Dir(targetPath); parent != "." && parent != "/" {
		if err := targetClient.MkdirAll(parent); err != nil {
			return 0, fmt.Errorf("create target parent %s: %w", parent, err)
		}
	}

	source, err := sourceClient.Open(sourcePath)
	if err != nil {
		return 0, fmt.Errorf("open source file %s: %w", sourcePath, err)
	}
	defer source.Close()

	written, err := uploadToPath(targetClient, targetPath, source)
	if err != nil {
		return 0, err
	}

	return written, nil
}

func copyRemoteDir(sourceClient *sftp.Client, targetClient *sftp.Client, sourcePath string, targetPath string) ([]TransferResult, []string, int64, error) {
	if err := targetClient.MkdirAll(targetPath); err != nil {
		return nil, nil, 0, fmt.Errorf("create target directory %s: %w", targetPath, err)
	}

	files := make([]TransferResult, 0)
	dirs := []string{targetPath}
	var totalSize int64

	items, err := sourceClient.ReadDir(sourcePath)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("read source directory %s: %w", sourcePath, err)
	}

	for _, item := range items {
		childSourcePath := path.Join(sourcePath, item.Name())
		childTargetPath := path.Join(targetPath, item.Name())
		if item.IsDir() {
			childFiles, childDirs, childSize, err := copyRemoteDir(sourceClient, targetClient, childSourcePath, childTargetPath)
			if err != nil {
				return nil, nil, 0, err
			}
			files = append(files, childFiles...)
			dirs = append(dirs, childDirs...)
			totalSize += childSize
			continue
		}

		written, err := copyRemoteFile(sourceClient, targetClient, childSourcePath, childTargetPath)
		if err != nil {
			return nil, nil, 0, err
		}
		files = append(files, TransferResult{RemotePath: childTargetPath, Size: written})
		totalSize += written
	}

	return files, dirs, totalSize, nil
}

func removeRemote(client *sftp.Client, remotePath string) error {
	if isProtectedDeletePath(remotePath) {
		return fmt.Errorf("refuse to remove protected path: %s", remotePath)
	}

	stat, err := client.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("stat remote path %s: %w", remotePath, err)
	}

	if !stat.IsDir() {
		if err := client.Remove(remotePath); err != nil {
			return fmt.Errorf("remove remote file %s: %w", remotePath, err)
		}
		return nil
	}

	items, err := client.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("read remote directory %s: %w", remotePath, err)
	}
	for _, item := range items {
		if err := removeRemote(client, path.Join(remotePath, item.Name())); err != nil {
			return err
		}
	}

	if err := client.RemoveDirectory(remotePath); err != nil {
		return fmt.Errorf("remove remote directory %s: %w", remotePath, err)
	}
	return nil
}

func isProtectedDeletePath(remotePath string) bool {
	cleaned := path.Clean(strings.TrimSpace(remotePath))
	return cleaned == "" || cleaned == "." || cleaned == "/"
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
