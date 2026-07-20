package sftpsvc

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"wiShell/backend/internal/model"
	"wiShell/backend/internal/sshsvc"
)

func RenameItem(conn model.Connection, remotePath string, newName string, timeout time.Duration) (RenameResult, error) {
	if err := validateRenameSourcePath(remotePath); err != nil {
		return RenameResult{}, err
	}
	if err := validateRenameName(newName); err != nil {
		return RenameResult{}, err
	}

	sshClient, err := sshsvc.SharedClient(conn, timeout)
	if err != nil {
		return RenameResult{}, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshsvc.DropSharedClient(conn)
		return RenameResult{}, fmt.Errorf("create sftp client: %w", err)
	}
	defer sftpClient.Close()

	resolvedSource, err := resolveRemotePath(sftpClient, remotePath)
	if err != nil {
		return RenameResult{}, fmt.Errorf("resolve rename source: %w", err)
	}
	targetPath, err := renameTargetPath(resolvedSource, newName)
	if err != nil {
		return RenameResult{}, err
	}

	stat, err := sftpClient.Lstat(resolvedSource)
	if err != nil {
		return RenameResult{}, fmt.Errorf("stat rename source %s: %w", resolvedSource, err)
	}

	result := RenameResult{
		OK:      true,
		OldPath: resolvedSource,
		NewPath: targetPath,
		Name:    newName,
		IsDir:   stat.IsDir(),
	}
	if targetPath == resolvedSource {
		return result, nil
	}

	exists, err := remotePathExists(sftpClient, targetPath)
	if err != nil {
		return RenameResult{}, err
	}
	if exists {
		return RenameResult{}, fmt.Errorf("%w: %s", ErrTargetExists, targetPath)
	}

	// SFTP Rename has no overwrite extension semantics. Do not use PosixRename,
	// because that extension explicitly replaces an existing destination.
	if err := sftpClient.Rename(resolvedSource, targetPath); err != nil {
		return RenameResult{}, fmt.Errorf("rename remote path %s to %s: %w", resolvedSource, targetPath, err)
	}

	result.Changed = true
	return result, nil
}

func validateRenameSourcePath(remotePath string) error {
	if strings.TrimSpace(remotePath) == "" {
		return fmt.Errorf("%w: empty path", ErrInvalidRenamePath)
	}

	cleaned := path.Clean(remotePath)
	if cleaned == "/" || cleaned == "~" {
		return fmt.Errorf("%w: %s", ErrProtectedRenamePath, remotePath)
	}
	if !strings.HasPrefix(cleaned, "/") && !strings.HasPrefix(cleaned, "~/") {
		return fmt.Errorf("%w: path must be absolute or under home: %s", ErrInvalidRenamePath, remotePath)
	}
	return nil
}

func validateRenameName(newName string) error {
	if newName == "" || strings.TrimSpace(newName) == "" {
		return fmt.Errorf("%w: name is empty", ErrInvalidRenameName)
	}
	if newName == "." || newName == ".." {
		return fmt.Errorf("%w: %s", ErrInvalidRenameName, newName)
	}
	if strings.ContainsRune(newName, '/') {
		return fmt.Errorf("%w: name must not contain '/'", ErrInvalidRenameName)
	}
	if strings.ContainsRune(newName, '\x00') {
		return fmt.Errorf("%w: name must not contain NUL", ErrInvalidRenameName)
	}
	return nil
}

func renameTargetPath(sourcePath string, newName string) (string, error) {
	if err := validateRenameName(newName); err != nil {
		return "", err
	}

	cleanedSource := path.Clean(sourcePath)
	if cleanedSource == "/" || cleanedSource == "~" {
		return "", fmt.Errorf("%w: %s", ErrProtectedRenamePath, sourcePath)
	}
	if !strings.HasPrefix(cleanedSource, "/") {
		return "", fmt.Errorf("%w: resolved path must be absolute: %s", ErrInvalidRenamePath, sourcePath)
	}
	return path.Join(path.Dir(cleanedSource), newName), nil
}
