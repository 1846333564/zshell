# WebSocket Terminal Module Status

## Scope

WebSocket bridge between xterm.js frontend and SSH PTY shell.

## Important Files

- `terminal_handler.go`
- `protocol.go`
- `terminal_handler_test.go`

## Current State

Supports raw input, output streaming, resize messages, ping/pong, and UTF-8 boundary handling.

## Known Work

Keep WebSocket on the real local API server; Wails asset server does not support terminal WebSocket upgrades.
