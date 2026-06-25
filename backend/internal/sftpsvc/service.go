package sftpsvc

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"zshell/backend/internal/model"
	"zshell/backend/internal/sshsvc"
)

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
