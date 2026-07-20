package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"wiShell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpRenameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ConnectionID) == "" || strings.TrimSpace(req.Path) == "" {
		writeError(w, http.StatusBadRequest, "connectionId and path are required")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	result, err := sftpsvc.RenameItem(conn, req.Path, req.NewName, s.sshTimeout)
	if err != nil {
		status := http.StatusBadGateway
		switch {
		case errors.Is(err, sftpsvc.ErrTargetExists):
			status = http.StatusConflict
		case errors.Is(err, sftpsvc.ErrInvalidRenameName),
			errors.Is(err, sftpsvc.ErrInvalidRenamePath),
			errors.Is(err, sftpsvc.ErrProtectedRenamePath):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
