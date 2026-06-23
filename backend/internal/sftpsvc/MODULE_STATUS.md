# SFTP Service Module Status

## Scope

Remote directory listing, uploads, downloads, zip archive generation, and server-to-server copy/move.

## Important Files

- `service.go`
- `service_test.go`

## Current State

Supports directory listing, multi-file/multi-folder uploads, recursive directory archive download, and recursive copy/move through SFTP. Directory listings resolve requested paths through SFTP `RealPath`, return the resolved path, include mode and UID:GID owner metadata, and sort folders before files.

## Known Work

Backend path APIs should continue to support both `~` and `/`. Real-server validation is still needed for deep trees, large transfers, permission-denied paths, and whether friendly owner names are worth resolving beyond UID:GID.
