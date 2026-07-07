package httpapi

import (
	"net/http"
	"strings"

	"wiShell/backend/internal/sshsvc"
)

func (s *Server) handleSSHTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req idRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	if err := sshsvc.TestConnection(conn, s.sshTimeout); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	hardware, err := sshsvc.ReadHardwareInfo(conn, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	conn.Hardware = hardware
	updated := s.store.Put(conn)
	if err := s.saveConnectionConfigs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"hardware":   hardware,
		"connection": updated.Summary(),
	})
}

func (s *Server) handleSSHExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req execRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(req.Command) == "" {
		writeError(w, http.StatusBadRequest, "command cannot be empty")
		return
	}

	conn, ok := s.store.Get(req.ConnectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	result, err := sshsvc.ExecCommand(conn, req.Command, s.sshTimeout)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}
