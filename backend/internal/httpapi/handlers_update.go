package httpapi

import (
	"encoding/json"
	"net/http"

	"zshell/backend/internal/logsvc"
	"zshell/backend/internal/updatesvc"
)

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	result, err := s.update.Check(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"update": result})
}

func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	result, err := s.update.Apply(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"update": result})
}

func (s *Server) handleUpdateApplyStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

	result, err := s.update.ApplyWithProgress(r.Context(), func(event updatesvc.ProgressEvent) {
		writeEvent(map[string]any{
			"type":     "progress",
			"progress": event,
		})
	})
	if err != nil {
		logsvc.Error("应用更新流式执行失败", err)
		writeEvent(map[string]any{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	writeEvent(map[string]any{
		"type":   "result",
		"update": result,
	})
}
