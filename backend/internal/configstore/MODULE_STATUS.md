# Config Store Module Status

## Scope

Windows-user encrypted saved connection configuration.

## Important Files

- `store.go`
- `dpapi_windows.go`
- `dpapi_other.go`

## Current State

Saved connections are written to `%AppData%\zShell\connections.dpapi` through `os.UserConfigDir()` and encrypted/decrypted with Windows DPAPI for the current user. Non-Windows builds return an explicit unsupported error.

## Known Work

Add unit coverage around load/save error handling with injectable paths or crypto wrappers if this module grows beyond the current Windows desktop target.
