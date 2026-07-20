package sftpsvc

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"wiShell/backend/internal/model"
	"wiShell/backend/internal/sshsvc"
)

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
		if action == "move" {
			noOp, err := ensureMoveTargetAvailable(sourcePath, targetPath, sameConnection, func(remotePath string) (bool, error) {
				return remotePathExists(targetSFTP, remotePath)
			})
			if err != nil {
				return TransferBatchResult{}, err
			}
			if noOp {
				appendTransferResult(&result, targetPath, stat)
				continue
			}
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
		if action == "move" {
			noOp, err := ensureMoveTargetAvailable(sourcePath, targetPath, true, func(remotePath string) (bool, error) {
				return remotePathExists(sftpClient, remotePath)
			})
			if err != nil {
				return TransferBatchResult{}, err
			}
			if noOp {
				appendTransferResult(&result, targetPath, stat)
				continue
			}
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
	if _, err := client.Lstat(remotePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat target path %s: %w", remotePath, err)
	}
	return true, nil
}

type remotePathExistsFunc func(string) (bool, error)

func ensureMoveTargetAvailable(sourcePath string, targetPath string, sameConnection bool, exists remotePathExistsFunc) (bool, error) {
	if sameConnection && targetPath == sourcePath {
		return true, nil
	}

	targetExists, err := exists(targetPath)
	if err != nil {
		return false, err
	}
	if targetExists {
		return false, fmt.Errorf("%w: %s", ErrTargetExists, targetPath)
	}
	return false, nil
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
	return fmt.Sprintf("mv -nT -- %s %s && if [ -e %s ] || [ -L %s ]; then echo 'move refused because target exists' >&2; exit 1; fi", sourceArg, targetArg, sourceArg, sourceArg)
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

	written, err := uploadToNewPath(targetClient, targetPath, source)
	if err != nil {
		return 0, err
	}

	return written, nil
}

func copyRemoteDir(sourceClient *sftp.Client, targetClient *sftp.Client, sourcePath string, targetPath string) ([]TransferResult, []string, int64, error) {
	if err := targetClient.Mkdir(targetPath); err != nil {
		if os.IsExist(err) {
			return nil, nil, 0, fmt.Errorf("%w: %s", ErrTargetExists, targetPath)
		}
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

func uploadToNewPath(client *sftp.Client, remotePath string, source io.Reader) (int64, error) {
	destination, err := client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		if os.IsExist(err) {
			return 0, fmt.Errorf("%w: %s", ErrTargetExists, remotePath)
		}
		return 0, fmt.Errorf("create remote file %s: %w", remotePath, err)
	}

	written, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		_ = client.Remove(remotePath)
		return 0, fmt.Errorf("copy to remote file %s: %w", remotePath, copyErr)
	}
	if closeErr != nil {
		_ = client.Remove(remotePath)
		return 0, fmt.Errorf("close remote file %s: %w", remotePath, closeErr)
	}
	return written, nil
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
