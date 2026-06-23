package sftpsvc

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
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
	Owner   string `json:"owner"`
	ModTime string `json:"modTime"`
}

type UploadItem struct {
	FileName     string
	RelativePath string
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
	client, err := sshsvc.NewClient(conn, timeout)
	if err != nil {
		return UploadBatchResult{}, err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
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

	for _, item := range files {
		targetRelative, err := cleanRelativePath(item.RelativePath, item.FileName)
		if err != nil {
			return UploadBatchResult{}, fmt.Errorf("invalid upload path %q: %w", item.RelativePath, err)
		}
		if targetRelative == "" {
			return UploadBatchResult{}, fmt.Errorf("invalid upload file name %q", item.FileName)
		}

		remotePath := path.Join(resolvedDir, targetRelative)
		if parent := path.Dir(remotePath); parent != "." && parent != "/" {
			if err := sftpClient.MkdirAll(parent); err != nil {
				return UploadBatchResult{}, fmt.Errorf("create remote parent %s: %w", parent, err)
			}
		}

		source, err := item.Open()
		if err != nil {
			return UploadBatchResult{}, fmt.Errorf("open upload source %s: %w", item.FileName, err)
		}

		written, uploadErr := uploadToPath(sftpClient, remotePath, source)
		closeErr := source.Close()
		if uploadErr != nil {
			return UploadBatchResult{}, uploadErr
		}
		if closeErr != nil {
			return UploadBatchResult{}, fmt.Errorf("close upload source %s: %w", item.FileName, closeErr)
		}

		result.Files = append(result.Files, UploadResult{RemotePath: remotePath, Size: written})
		result.TotalSize += written
	}

	return result, nil
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

	sourceSSH, err := sshsvc.NewClient(sourceConn, timeout)
	if err != nil {
		return TransferBatchResult{}, err
	}
	defer sourceSSH.Close()

	targetSSH, err := sshsvc.NewClient(targetConn, timeout)
	if err != nil {
		return TransferBatchResult{}, err
	}
	defer targetSSH.Close()

	sourceSFTP, err := sftp.NewClient(sourceSSH)
	if err != nil {
		return TransferBatchResult{}, fmt.Errorf("create source sftp client: %w", err)
	}
	defer sourceSFTP.Close()

	targetSFTP, err := sftp.NewClient(targetSSH)
	if err != nil {
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
	dst, err := client.Create(remotePath)
	if err != nil {
		return 0, fmt.Errorf("create remote file %s: %w", remotePath, err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, source)
	if err != nil {
		return 0, fmt.Errorf("upload copy to %s: %w", remotePath, err)
	}

	return written, nil
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
