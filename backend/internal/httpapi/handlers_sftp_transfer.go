package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"wiShell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.SourceConnectionID) == "" || strings.TrimSpace(req.TargetConnectionID) == "" {
		writeError(w, http.StatusBadRequest, "sourceConnectionId and targetConnectionId are required")
		return
	}
	if strings.TrimSpace(req.TargetPath) == "" {
		writeError(w, http.StatusBadRequest, "targetPath is required")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "items are required")
		return
	}

	sourceConn, ok := s.store.Get(req.SourceConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "source connection not found")
		return
	}
	targetConn, ok := s.store.Get(req.TargetConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "target connection not found")
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "copy"
	}

	result, err := sftpsvc.TransferItems(sourceConn, targetConn, req.TargetPath, req.Items, action, s.sshTimeout)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, sftpsvc.ErrTargetExists) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSFTPDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.ConnectionID) == "" {
		writeError(w, http.StatusBadRequest, "connectionId is required")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "items are required")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	result, err := sftpsvc.DeleteItems(conn, req.Items, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
