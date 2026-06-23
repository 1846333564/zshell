# WebSocket Terminal Module Status

## Scope

WebSocket bridge between xterm.js frontend and SSH PTY shell.

## Important Files

- `terminal_handler.go`
- `protocol.go`
- `terminal_handler_test.go`

## Current State

Supports raw input, output streaming, resize messages, protocol ping/pong, server-side WebSocket ping frames with read deadlines, and UTF-8 boundary handling.

## Known Work

Keep WebSocket on the real local API server; Wails asset server does not support terminal WebSocket upgrades. If users still see idle disconnects, capture the terminal error code before close to distinguish WebSocket transport failure from remote SSH shell exit.
