package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"zshell/backend/internal/logsvc"
	"zshell/backend/internal/sftpsvc"
)

func (s *Server) handleSFTPUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("parse multipart: %v", err))
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	connectionID := strings.TrimSpace(r.FormValue("connectionId"))
	remoteDir := strings.TrimSpace(r.FormValue("path"))
	if connectionID == "" {
		writeError(w, http.StatusBadRequest, "connectionId is required")
		return
	}

	conn, ok := s.store.Get(connectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	files, directories := multipartUploadItems(r.MultipartForm)
	if len(files) == 0 && len(directories) == 0 {
		writeError(w, http.StatusBadRequest, "at least one file or directory is required")
		return
	}

	result, err := sftpsvc.UploadFiles(conn, remoteDir, files, directories, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	payload := map[string]any{
		"ok":          result.OK,
		"files":       result.Files,
		"directories": result.Directories,
		"totalSize":   result.TotalSize,
	}
	if len(result.Files) == 1 {
		payload["remotePath"] = result.Files[0].RemotePath
		payload["size"] = result.Files[0].Size
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleSFTPUploadStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("parse multipart: %v", err))
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	connectionID := strings.TrimSpace(r.FormValue("connectionId"))
	remoteDir := strings.TrimSpace(r.FormValue("path"))
	if connectionID == "" {
		writeError(w, http.StatusBadRequest, "connectionId is required")
		return
	}

	conn, ok := s.store.Get(connectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	files, directories := multipartUploadItems(r.MultipartForm)
	if len(files) == 0 && len(directories) == 0 {
		writeError(w, http.StatusBadRequest, "at least one file or directory is required")
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

	result, err := sftpsvc.UploadFilesWithProgress(conn, remoteDir, files, directories, s.sshTimeout, func(event sftpsvc.UploadProgressEvent) {
		writeEvent(map[string]any{
			"type":     "progress",
			"progress": event,
		})
	})
	if err != nil {
		logsvc.Error("SFTP 流式上传失败", err)
		writeEvent(map[string]any{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	payload := map[string]any{
		"ok":          result.OK,
		"files":       result.Files,
		"directories": result.Directories,
		"totalSize":   result.TotalSize,
	}
	if len(result.Files) == 1 {
		payload["remotePath"] = result.Files[0].RemotePath
		payload["size"] = result.Files[0].Size
	}
	writeEvent(map[string]any{
		"type":   "result",
		"upload": payload,
	})
}
