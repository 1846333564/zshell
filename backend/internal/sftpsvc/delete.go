package sftpsvc

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"wiShell/backend/internal/model"
	"wiShell/backend/internal/sshsvc"
)

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

func isProtectedDeletePath(remotePath string) bool {
	cleaned := path.Clean(strings.TrimSpace(remotePath))
	return cleaned == "" || cleaned == "." || cleaned == "/"
}
