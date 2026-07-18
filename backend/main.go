package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"wiShell/backend/internal/appinfo"
	"wiShell/backend/internal/httpapi"
	"wiShell/backend/internal/logsvc"
	"wiShell/backend/internal/store"
	"wiShell/backend/internal/web"
)

func main() {
	logger, err := logsvc.InitDefault()
	if err != nil {
		log.Printf("init log system failed: %v", err)
	} else {
		defer logger.Close()
	}
	defer logsvc.RecoverAndExit("Wails 桌面入口")
	log.Printf("wiShell 启动，版本：%s", appinfo.Version)

	connectionStore := store.NewMemoryStore()
	apiServer := httpapi.NewServer(connectionStore, 10*time.Second)
	gpuAccelerationEnabled, err := apiServer.GPUAccelerationEnabled()
	if err != nil {
		gpuAccelerationEnabled = true
		log.Printf("load GPU acceleration preference failed, using enabled default: %v", err)
	}
	log.Printf("WebView2 GPU acceleration enabled: %t", gpuAccelerationEnabled)

	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	listener, port, err := listenOnDynamicPort()
	if err != nil {
		log.Fatalf("bind local port failed: %v", err)
	}

	server := &http.Server{
		Handler:           httpapi.WithCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		defer logsvc.Recover("本地 API 服务 goroutine")
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("server failed: %v", err)
		}
	}()

	apiBaseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	log.Printf("wiShell API listening on %s", apiBaseURL)

	err = wails.Run(&options.App{
		Title:                    "wiShell",
		Width:                    1480,
		Height:                   920,
		MinWidth:                 1180,
		MinHeight:                760,
		WindowStartState:         options.Normal,
		BackgroundColour:         options.NewRGB(5, 12, 18),
		AssetServer:              &assetserver.Options{Handler: web.HandlerWithConfig(apiBaseURL)},
		OnShutdown:               shutdownServer(server),
		Frameless:                true,
		EnableDefaultContextMenu: true,
		Windows: &windows.Options{
			Theme: windows.Dark,
			// Honor the stored preference while defaulting to hardware acceleration.
			// Software rasterisation can make even empty-state interaction visibly lag.
			WebviewGpuIsDisabled: !gpuAccelerationEnabled,
			IsZoomControlEnabled: true,
			DisablePinchZoom:     false,
			ResizeDebounceMS:     120,
			DLLSearchPaths:       windows.DLLSearchDefaultDirs,
			WebviewUserDataPath:  "",
			WebviewBrowserPath:   "",
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			EnableSwipeGestures:  true,
			BackdropType:         windows.None,
		},
	})
	if err != nil {
		log.Fatalf("wails failed: %v", err)
	}
}

func shutdownServer(server *http.Server) func(context.Context) {
	return func(ctx context.Context) {
		shutdownCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
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
