package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"zshell/backend/internal/model"
	"zshell/backend/internal/sftpsvc"
	"zshell/backend/internal/sshsvc"
	"zshell/backend/internal/store"
	"zshell/backend/internal/ws"
)

type Server struct {
	store      *store.MemoryStore
	sshTimeout time.Duration
	terminalWS *ws.TerminalHandler
}

func NewServer(connStore *store.MemoryStore, sshTimeout time.Duration) *Server {
	return &Server{
		store:      connStore,
		sshTimeout: sshTimeout,
		terminalWS: ws.NewTerminalHandler(connStore, sshTimeout),
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/connections", s.handleConnections)
	mux.HandleFunc("/api/ssh/test", s.handleSSHTest)
	mux.HandleFunc("/api/ssh/exec", s.handleSSHExec)
	mux.HandleFunc("/api/sftp/list", s.handleSFTPList)
	mux.HandleFunc("/api/sftp/upload", s.handleSFTPUpload)
	mux.HandleFunc("/api/sftp/download", s.handleSFTPDownload)
	mux.HandleFunc("/api/sftp/archive", s.handleSFTPArchive)
	mux.HandleFunc("/api/sftp/transfer", s.handleSFTPTransfer)
	mux.Handle("/ws/terminal", s.terminalWS)
}

type createConnectionRequest struct {
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

type sftpTransferRequest struct {
	SourceConnectionID string                 `json:"sourceConnectionId"`
	TargetConnectionID string                 `json:"targetConnectionId"`
	TargetPath         string                 `json:"targetPath"`
	Action             string                 `json:"action"`
	Items              []sftpsvc.TransferItem `json:"items"`
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

func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"connections": s.store.List()})
	case http.MethodPost:
		var req createConnectionRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := validateConnectionRequest(req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if req.Port == 0 {
			req.Port = 22
		}
		authMethod := normalizeAuthMethod(req.AuthMethod)

		conn := model.Connection{
			Name:       req.Name,
			Host:       req.Host,
			Port:       req.Port,
			Username:   req.Username,
			Password:   req.Password,
			AuthMethod: authMethod,
		}

		created := s.store.Add(conn)
		writeJSON(w, http.StatusCreated, map[string]any{"connection": created.Summary()})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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

func validateConnectionRequest(req createConnectionRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.Host) == "" {
		return errors.New("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("username is required")
	}
	authMethod := normalizeAuthMethod(req.AuthMethod)
	if authMethod != "password" && authMethod != "id_rsa" {
		return errors.New("authMethod must be password or id_rsa")
	}
	if authMethod == "password" && strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}
	return nil
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
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
