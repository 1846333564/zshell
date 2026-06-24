package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"zshell/backend/internal/appinfo"
	"zshell/backend/internal/configstore"
	"zshell/backend/internal/model"
	"zshell/backend/internal/monitorsvc"
	"zshell/backend/internal/sftpsvc"
	"zshell/backend/internal/sshsvc"
	"zshell/backend/internal/store"
	"zshell/backend/internal/updatesvc"
	"zshell/backend/internal/ws"
)

type Server struct {
	store       *store.MemoryStore
	configStore *configstore.Store
	sshTimeout  time.Duration
	terminalWS  *ws.TerminalHandler
	monitor     *monitorsvc.Service
	update      *updatesvc.Service
}

func NewServer(connStore *store.MemoryStore, sshTimeout time.Duration) *Server {
	cfgStore, err := configstore.NewDefault()
	if err != nil {
		log.Printf("connection config store unavailable: %v", err)
	} else {
		connections, err := cfgStore.Load()
		if err != nil {
			log.Printf("load connection config failed: %v", err)
		} else {
			for _, conn := range connections {
				connStore.Put(conn)
			}
		}
	}

	return &Server{
		store:       connStore,
		configStore: cfgStore,
		sshTimeout:  sshTimeout,
		terminalWS:  ws.NewTerminalHandler(connStore, sshTimeout),
		monitor:     monitorsvc.NewService(),
		update:      updatesvc.NewService(),
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/app/info", s.handleAppInfo)
	mux.HandleFunc("/api/connections", s.handleConnections)
	mux.HandleFunc("/api/config/connections", s.handleConnectionConfigs)
	mux.HandleFunc("/api/config/preferences", s.handleUIPreferences)
	mux.HandleFunc("/api/update/check", s.handleUpdateCheck)
	mux.HandleFunc("/api/update/apply", s.handleUpdateApply)
	mux.HandleFunc("/api/update/apply/stream", s.handleUpdateApplyStream)
	mux.HandleFunc("/api/ssh/test", s.handleSSHTest)
	mux.HandleFunc("/api/ssh/exec", s.handleSSHExec)
	mux.HandleFunc("/api/sftp/list", s.handleSFTPList)
	mux.HandleFunc("/api/sftp/upload", s.handleSFTPUpload)
	mux.HandleFunc("/api/sftp/download", s.handleSFTPDownload)
	mux.HandleFunc("/api/sftp/file/read", s.handleSFTPFileRead)
	mux.HandleFunc("/api/sftp/file/write", s.handleSFTPFileWrite)
	mux.HandleFunc("/api/sftp/archive", s.handleSFTPArchive)
	mux.HandleFunc("/api/sftp/transfer", s.handleSFTPTransfer)
	mux.HandleFunc("/api/sftp/delete", s.handleSFTPDelete)
	mux.HandleFunc("/api/monitor/snapshot", s.handleMonitorSnapshot)
	mux.Handle("/ws/terminal", s.terminalWS)
}

type createConnectionRequest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	AuthMethod string `json:"authMethod"`
}

type idRequest struct {
	ConnectionID string `json:"connectionId"`
}

type execRequest struct {
	ConnectionID string `json:"connectionId"`
	Command      string `json:"command"`
}

type sftpListRequest struct {
	ConnectionID string `json:"connectionId"`
	Path         string `json:"path"`
}

type sftpFileReadRequest struct {
	ConnectionID string `json:"connectionId"`
	Path         string `json:"path"`
}

type sftpFileWriteRequest struct {
	ConnectionID string `json:"connectionId"`
	Path         string `json:"path"`
	Content      string `json:"content"`
}

type sftpTransferRequest struct {
	SourceConnectionID string                 `json:"sourceConnectionId"`
	TargetConnectionID string                 `json:"targetConnectionId"`
	TargetPath         string                 `json:"targetPath"`
	Action             string                 `json:"action"`
	Items              []sftpsvc.TransferItem `json:"items"`
}

type sftpDeleteRequest struct {
	ConnectionID string                 `json:"connectionId"`
	Items        []sftpsvc.TransferItem `json:"items"`
}

type monitorSnapshotRequest struct {
	ConnectionID string `json:"connectionId"`
	ProcessSort  string `json:"processSort"`
}

type uiPreferencesRequest struct {
	UIScale          *float64 `json:"uiScale"`
	TerminalFontSize *int     `json:"terminalFontSize"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleAppInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"app": appinfo.Current()})
}

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

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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

func (s *Server) handleSFTPArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	connectionID := strings.TrimSpace(r.URL.Query().Get("connectionId"))
	remotePaths := r.URL.Query()["path"]
	if connectionID == "" || len(remotePaths) == 0 {
		writeError(w, http.StatusBadRequest, "connectionId and at least one path are required")
		return
	}

	conn, ok := s.store.Get(connectionID)
	if !ok {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", archiveName(remotePaths)))
	if err := sftpsvc.ArchiveItems(conn, remotePaths, w, s.sshTimeout); err != nil {
		return
	}
}

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
		writeError(w, http.StatusBadGateway, err.Error())
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

func connectionFromRequest(req createConnectionRequest, existing model.Connection) model.Connection {
	authMethod := normalizeAuthMethod(req.AuthMethod)
	password := strings.TrimSpace(req.Password)
	if authMethod == "password" && password == "" {
		password = existing.Password
	}
	if authMethod != "password" {
		password = ""
	}

	id := strings.TrimSpace(existing.ID)
	if id == "" {
		id = strings.TrimSpace(req.ID)
	}

	return model.Connection{
		ID:         id,
		Name:       strings.TrimSpace(req.Name),
		Host:       strings.TrimSpace(req.Host),
		Port:       req.Port,
		Username:   strings.TrimSpace(req.Username),
		Password:   password,
		AuthMethod: authMethod,
	}
}

func validateConnectionRequest(req createConnectionRequest, existing model.Connection) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.Host) == "" {
		return errors.New("host is required")
	}
	if req.Port < 1 || req.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("username is required")
	}
	authMethod := normalizeAuthMethod(req.AuthMethod)
	if authMethod != "password" && authMethod != "id_rsa" {
		return errors.New("authMethod must be password or id_rsa")
	}
	if authMethod == "password" && strings.TrimSpace(req.Password) == "" && strings.TrimSpace(existing.Password) == "" {
		return errors.New("password is required")
	}
	return nil
}

func (s *Server) saveConnectionConfigs() error {
	if s.configStore == nil {
		return errors.New("connection config store unavailable")
	}
	if err := s.configStore.Save(s.store.ListFull()); err != nil {
		return err
	}
	return nil
}

func (s *Server) loadUIPreferences() (configstore.Preferences, error) {
	if s.configStore == nil {
		return configstore.Preferences{}, errors.New("connection config store unavailable")
	}
	preferences, err := s.configStore.LoadPreferences()
	if err != nil {
		return configstore.Preferences{}, err
	}
	preferences.UIScale = normalizeUIScale(preferences.UIScale)
	preferences.TerminalFontSize = normalizeTerminalFontSize(preferences.TerminalFontSize)
	return preferences, nil
}

func (s *Server) saveUIPreferences(preferences configstore.Preferences) error {
	if s.configStore == nil {
		return errors.New("connection config store unavailable")
	}
	preferences.UIScale = normalizeUIScale(preferences.UIScale)
	preferences.TerminalFontSize = normalizeTerminalFontSize(preferences.TerminalFontSize)
	return s.configStore.SavePreferences(preferences)
}

func normalizeUIScale(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 1
	}
	value = math.Min(1.35, math.Max(0.82, value))
	return math.Round(value*100) / 100
}

func normalizeTerminalFontSize(value int) int {
	if value <= 0 {
		return 14
	}
	if value < 10 {
		return 10
	}
	if value > 28 {
		return 28
	}
	return value
}

func normalizeAuthMethod(value string) string {
	authMethod := strings.ToLower(strings.TrimSpace(value))
	if authMethod == "" {
		return "password"
	}
	return authMethod
}

func multipartUploadItems(form *multipart.Form) ([]sftpsvc.UploadItem, []string) {
	if form == nil {
		return nil, nil
	}

	fileHeaders := make([]*multipart.FileHeader, 0)
	fileHeaders = append(fileHeaders, form.File["files"]...)
	fileHeaders = append(fileHeaders, form.File["file"]...)

	relativePaths := form.Value["relativePaths"]
	files := make([]sftpsvc.UploadItem, 0, len(fileHeaders))
	for index, header := range fileHeaders {
		header := header
		relativePath := ""
		if index < len(relativePaths) {
			relativePath = relativePaths[index]
		}
		files = append(files, sftpsvc.UploadItem{
			FileName:     header.Filename,
			RelativePath: relativePath,
			Open: func() (io.ReadCloser, error) {
				return header.Open()
			},
		})
	}

	return files, form.Value["directories"]
}

func archiveName(remotePaths []string) string {
	if len(remotePaths) == 1 {
		base := path.Base(strings.TrimSpace(remotePaths[0]))
		if base != "." && base != "/" && base != "" {
			return base + ".zip"
		}
	}
	return "zshell-download.zip"
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
