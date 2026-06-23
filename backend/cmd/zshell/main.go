package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"zshell/backend/internal/httpapi"
	"zshell/backend/internal/store"
	"zshell/backend/internal/web"
)

func main() {
	connectionStore := store.NewMemoryStore()
	apiServer := httpapi.NewServer(connectionStore, 10*time.Second)

	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)
	mux.Handle("/", web.Handler())

	listener, port, err := listenOnDynamicPort()
	if err != nil {
		log.Fatalf("bind local port failed: %v", err)
	}

	server := &http.Server{
		Handler:      httpapi.WithCORS(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	log.Printf("zShell listening on %s", url)
	openBrowser(url)

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func listenOnDynamicPort() (net.Listener, int, error) {
	for attempt := 0; attempt < 200; attempt++ {
		port := randomHighPort()
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return listener, port, nil
		}
	}

	return nil, 0, fmt.Errorf("no free local port found above 10000")
}

func randomHighPort() int {
	var buf [2]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 10001 + int(time.Now().UnixNano()%55535)
	}
	return 10001 + int(binary.BigEndian.Uint16(buf[:])%55535)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
