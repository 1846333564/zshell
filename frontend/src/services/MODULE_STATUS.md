# Frontend Services Module Status

## Scope

Client wrappers for HTTP API, downloads, uploads, monitor snapshots, transfer, and terminal WebSocket.

## Important Files

- `apiClient.js`
- `wsClient.js`

## Current State

API and WebSocket calls use the injected backend base URL in Wails or relative paths during Vite development. `apiClient.js` includes saved connection config wrappers for list/create/update/delete.

## Known Work

Keep browser storage out of saved connection credentials; only non-secret transient UI state should use local browser storage.
