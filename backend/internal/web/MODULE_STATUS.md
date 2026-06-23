# Web Asset Module Status

## Scope

Embedded frontend asset serving and runtime backend base URL injection.

## Important Files

- `server.go`
- `app/.gitkeep`

## Current State

The Wails window loads Vue assets through `web.HandlerWithConfig`, which injects `window.__ZSHELL_BACKEND_BASE__` for HTTP and WebSocket calls.

## Known Work

Do not serve terminal WebSocket through Wails asset server.
