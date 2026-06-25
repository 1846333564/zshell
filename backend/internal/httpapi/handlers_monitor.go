package httpapi

import (
	"net/http"
)

func (s *Server) handleMonitorSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req monitorSnapshotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	snapshot, err := s.monitor.Snapshot(conn, req.ProcessSort, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"snapshot": snapshot})
}
