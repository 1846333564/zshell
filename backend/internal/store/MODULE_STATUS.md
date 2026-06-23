# Store Module Status

## Scope

In-memory active connection registry used by HTTP, terminal, SFTP, and monitor handlers.

## Important Files

- `memory_store.go`

## Current State

Connections are stored in memory by generated IDs or preserved IDs from saved config. `List` returns summaries without passwords; `ListFull` is used only for encrypted persistence. Listing is stable-sorted by name, host, then ID.

## Known Work

If short-lived unsaved sessions are reintroduced, split runtime sessions from persisted saved configs so temporary connections are not written to disk.
