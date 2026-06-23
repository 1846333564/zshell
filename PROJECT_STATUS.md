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
- File manager path navigation with fixed root `/`, resolved home paths such as `/root`, collapsible path navigation, and a refresh-only context menu.

## Active Gaps

- More complete real-server validation still depends on user SSH targets.
- Upload/copy/move/download-selection UI controls are currently not exposed in the file manager after the fixed toolbar was removed; backend endpoints still exist.

## Required Workflow

Before future changes, read this file and the relevant `MODULE_STATUS.md` files under backend/frontend modules.
