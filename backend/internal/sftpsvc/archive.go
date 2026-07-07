package sftpsvc

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"wiShell/backend/internal/model"
	"wiShell/backend/internal/sshsvc"
)

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
