package httpapi

import (
	"net/http"

	"wiShell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpListRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	resolved, entries, err := sftpsvc.ListDirectory(conn, req.Path, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":    resolved,
		"entries": entries,
	})
}
