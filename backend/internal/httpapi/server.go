package httpapi

import (
	"log"
	"net/http"
	"time"

	"wiShell/backend/internal/configstore"
	"wiShell/backend/internal/monitorsvc"
	"wiShell/backend/internal/store"
	"wiShell/backend/internal/updatesvc"
	"wiShell/backend/internal/ws"
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
	mux.HandleFunc("/api/sftp/upload/stream", s.handleSFTPUploadStream)
	mux.HandleFunc("/api/sftp/download", s.handleSFTPDownload)
	mux.HandleFunc("/api/sftp/file/read", s.handleSFTPFileRead)
	mux.HandleFunc("/api/sftp/file/read/raw", s.handleSFTPFileReadRaw)
	mux.HandleFunc("/api/sftp/file/read/stream", s.handleSFTPFileReadStream)
	mux.HandleFunc("/api/sftp/file/write", s.handleSFTPFileWrite)
	mux.HandleFunc("/api/sftp/archive", s.handleSFTPArchive)
	mux.HandleFunc("/api/sftp/transfer", s.handleSFTPTransfer)
	mux.HandleFunc("/api/sftp/rename", s.handleSFTPRename)
	mux.HandleFunc("/api/sftp/delete", s.handleSFTPDelete)
	mux.HandleFunc("/api/monitor/snapshot", s.handleMonitorSnapshot)
	mux.Handle("/ws/terminal", s.terminalWS)
}
