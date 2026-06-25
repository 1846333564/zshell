package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"zshell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPFileRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpFileReadRequest
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

	file, err := sftpsvc.ReadTextFile(conn, req.Path, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}

func (s *Server) handleSFTPFileReadStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpFileReadRequest
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	writeEvent := func(payload map[string]any) {
		_ = encoder.Encode(payload)
		flusher.Flush()
	}

	file, err := sftpsvc.ReadTextFileWithProgress(conn, req.Path, s.sshTimeout, func(event sftpsvc.TextReadProgressEvent) {
		writeEvent(map[string]any{
			"type":     "progress",
			"progress": event,
		})
	})
	if err != nil {
		writeEvent(map[string]any{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	writeEvent(map[string]any{
		"type": "result",
		"file": file,
	})
}

func (s *Server) handleSFTPFileWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req sftpFileWriteRequest
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

	file, err := sftpsvc.WriteTextFile(conn, req.Path, req.Content, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}
