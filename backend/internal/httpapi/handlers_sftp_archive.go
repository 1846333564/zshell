package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"zshell/backend/internal/logsvc"
	"zshell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	connectionID := strings.TrimSpace(r.URL.Query().Get("connectionId"))
	remotePaths := r.URL.Query()["path"]
	if connectionID == "" || len(remotePaths) == 0 {
		writeError(w, http.StatusBadRequest, "connectionId and at least one path are required")
		return
	}

	conn, ok := s.store.Get(connectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", archiveName(remotePaths)))
	if err := sftpsvc.ArchiveItems(conn, remotePaths, w, s.sshTimeout); err != nil {
		logsvc.Error("SFTP 归档下载失败", err)
		return
	}
}
