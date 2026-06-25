package sftpsvc

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"
)

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
