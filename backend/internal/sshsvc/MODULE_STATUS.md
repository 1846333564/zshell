# SSH Service Module Status

## Scope

SSH client creation, auth method selection, one-shot command execution, and interactive PTY shell.

## Important Files

- `client.go`
- `shell.go`

## Current State

Supports password auth and current Windows user's `~/.ssh/id_rsa` private key. PTY shell powers the terminal WebSocket.

## Known Work

Host key checking is still permissive. Private key mode is intentionally limited to `~/.ssh/id_rsa`.
