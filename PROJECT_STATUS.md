# zShell Project Status

## Overview

zShell is a Windows desktop SSH/SFTP tool built with Go, Wails/WebView2, Vue, xterm.js, and SFTP over SSH. The release artifact is `D:\zshell.exe`.

## Current Architecture

- `backend/main.go` is the Wails desktop entry. It starts a local dynamic high-port HTTP API/WebSocket service on `127.0.0.1` and opens the Vue app inside a native WebView2 window.
- `backend/cmd/zshell/main.go` remains a legacy local HTTP entry for development and does not launch the browser.
- `backend/internal/httpapi` exposes connection, SSH, SFTP, transfer, archive, and monitor endpoints.
- `backend/internal/configstore` persists saved connection configuration in the current Windows user's config directory with DPAPI encryption.
- `frontend/src/App.vue` owns the two-column desktop shell: left monitor panel, right connection tabs, terminal, and file area.
- `build-windows.ps1` is the release build path.

## Implemented

- Password and `~/.ssh/id_rsa` SSH authentication.
- Interactive PTY terminal over WebSocket.
- SFTP browse, upload, download, archive download, and remote copy/move.
- Wails Windows executable packaging.
- Linux monitor snapshot API and left-side monitor UI.
- Dynamic backend port above 10000.
- Backend-owned saved connection create/list/update/delete APIs with Windows DPAPI-encrypted storage.
- Saved connection editing in the frontend.
- Connection tabs that show only the connection name.
- File manager path navigation with fixed root `/`, resolved home paths such as `/root`, basename-only tree labels, centered right-side fold controls, color-only opened markers, full file context menu actions, and resizable file columns.
- File upload by picker or drag/drop shows a compact progress panel with total progress, per-file progress, upload speed, and auto-close after completion.
- Terminal-focused `Ctrl +` / `Ctrl -` font sizing, persisted non-terminal UI zoom, and terminal `Ctrl+Shift+C` / `Ctrl+Shift+V` clipboard shortcuts.
- Interactive terminals use SSH keepalive and server-side WebSocket ping/pong to reduce idle/background disconnects.
- The Wails window is frameless with a custom zShell top bar, placeholder `配置管理` and `UI管理` menus, custom window controls, and application-matched scrollbars.

## Active Gaps

- More complete real-server validation still depends on user SSH targets.
- SFTP owner display uses protocol UID:GID values; resolving friendly usernames would require extra remote lookup logic.

## Required Workflow

Before future changes, read this file and the relevant `MODULE_STATUS.md` files under backend/frontend modules.
