package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"wiShell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	connectionID := strings.TrimSpace(r.URL.Query().Get("connectionId"))
	remotePath := strings.TrimSpace(r.URL.Query().Get("path"))
	if connectionID == "" || remotePath == "" {
		writeError(w, http.StatusBadRequest, "connectionId and path are required")
		return
	}

	conn, ok := s.store.Get(connectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	stream, fileName, size, err := sftpsvc.DownloadFile(conn, remotePath, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(fileName)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))

	_, _ = io.Copy(w, stream)
}
