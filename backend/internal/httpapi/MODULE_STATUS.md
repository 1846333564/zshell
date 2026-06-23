# HTTP API Module Status

## Scope

HTTP routes for health, connection lifecycle, SSH actions, SFTP actions, cross-server transfer, archive download, and monitor snapshots.

## Important Files

- `server.go`

## Current State

Routes are registered on a standard `http.ServeMux`. Connection config routes live at `/api/config/connections` with `GET`, `POST`, `PUT`, and `DELETE`; UI preferences live at `/api/config/preferences` with `GET` and `PUT`. Saved configs are backed by `configstore` and active runtime lookups use `store.MemoryStore`. Editing a password connection with an empty password keeps the previously saved password. SFTP upload supports multiple files/directories. Monitor snapshots are returned by `POST /api/monitor/snapshot`.

## Known Work

Add focused handler tests for config create/update/delete, especially password retention and config-store failures.
