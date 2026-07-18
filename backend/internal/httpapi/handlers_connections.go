package httpapi

import (
	"net/http"
	"strings"

	"wiShell/backend/internal/model"
)

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"connections": s.store.List()})
	case http.MethodPost:
		s.createConnectionConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleConnectionConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"connections": s.store.List()})
	case http.MethodPost:
		s.createConnectionConfig(w, r)
	case http.MethodPut:
		s.updateConnectionConfig(w, r)
	case http.MethodDelete:
		s.deleteConnectionConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleUIPreferences(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		preferences, err := s.loadUIPreferences()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"preferences": preferences})
	case http.MethodPut:
		var req uiPreferencesRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		preferences, err := s.loadUIPreferences()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if req.UIScale != nil {
			preferences.UIScale = normalizeUIScale(*req.UIScale)
		}
		if req.TerminalFontSize != nil {
			preferences.TerminalFontSize = normalizeTerminalFontSize(*req.TerminalFontSize)
		}
		if req.ThemeKey != nil {
			preferences.ThemeKey = normalizeThemeKey(*req.ThemeKey)
		}
		if req.CustomTheme != nil {
			preferences.CustomTheme = normalizeCustomTheme(req.CustomTheme)
		}
		if req.GPUAccelerationEnabled != nil {
			preferences.GPUAccelerationEnabled = boolPointer(*req.GPUAccelerationEnabled)
		}
		if err := s.saveUIPreferences(preferences); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"preferences": preferences})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) createConnectionConfig(w http.ResponseWriter, r *http.Request) {
	var req createConnectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.ID = ""

	if err := validateConnectionRequest(req, model.Connection{}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	conn := connectionFromRequest(req, model.Connection{})
	created := s.store.Put(conn)
	if err := s.saveConnectionConfigs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"connection": created.Summary()})
}

func (s *Server) updateConnectionConfig(w http.ResponseWriter, r *http.Request) {
	var req createConnectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	existing, ok := s.store.Get(req.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection config not found")
		return
	}

	if err := validateConnectionRequest(req, existing); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated := s.store.Put(connectionFromRequest(req, existing))
	if err := s.saveConnectionConfigs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"connection": updated.Summary()})
}

func (s *Server) deleteConnectionConfig(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if !s.store.Delete(id) {
		writeError(w, http.StatusNotFound, "connection config not found")
		return
	}
	if err := s.saveConnectionConfigs(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
