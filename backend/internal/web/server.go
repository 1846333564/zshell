package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:app
var assets embed.FS

func Handler() http.Handler {
	return HandlerWithConfig("")
}

func HandlerWithConfig(apiBaseURL string) http.Handler {
	app, err := fs.Sub(assets, "app")
	if err != nil {
		return http.NotFoundHandler()
	}

	files := http.FileServer(http.FS(app))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath != "" {
			if _, err := fs.Stat(app, cleanPath); err == nil {
				files.ServeHTTP(w, r)
				return
			}
		}

		serveIndex(w, app, apiBaseURL)
	})
}

func serveIndex(w http.ResponseWriter, dist fs.FS, apiBaseURL string) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!doctype html><title>wiShell</title><body>wiShell frontend is not embedded. Run the Windows build script first.</body>"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(injectRuntimeConfig(index, apiBaseURL))
}

func injectRuntimeConfig(index []byte, apiBaseURL string) []byte {
	if strings.TrimSpace(apiBaseURL) == "" {
		return index
	}

	encoded, _ := json.Marshal(apiBaseURL)
	script := fmt.Sprintf("<script>window.__WISHELL_BACKEND_BASE__=%s;</script>", encoded)
	content := string(index)
	if strings.Contains(content, "</head>") {
		content = strings.Replace(content, "</head>", script+"</head>", 1)
	} else {
		content = script + content
	}
	return []byte(content)
}
