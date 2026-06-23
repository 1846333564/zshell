# Backend Module Status

## Scope

Go backend for Wails app startup, local API service, dynamic port binding, and release build integration.

## Important Files

- `main.go`: Wails desktop entry.
- `cmd/zshell/main.go`: legacy HTTP-only entry.
- `wails.json`: Wails project config.
- `go.mod`: backend dependencies.

## Current State

The backend starts local API/WebSocket services on dynamic high ports and serves the frontend through Wails. The Wails app uses a frameless Windows window, while WebView context-menu events remain enabled so Vue can render custom file-manager menus and suppress non-file-area menus. Saved connection configs are loaded during API server startup and stored in memory for runtime terminal/SFTP/monitor use.

## Known Work

Real-server smoke testing is still needed for saved password edits, terminal login, and SFTP navigation against the user's target servers.
