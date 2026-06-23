# Frontend Module Status

## Scope

Vue/Vite frontend for the Wails desktop window.

## Important Files

- `src/App.vue`
- `src/style.css`
- `src/components`
- `src/services`

## Current State

The UI is a two-column desktop layout with a custom frameless-window top bar, left monitor panel, and right connection tabs, terminal, and file area. Saved connection management now uses backend config APIs instead of browser-local password history. The app shell blocks non-file-area context menus while allowing file-manager custom context menus, persists non-terminal UI zoom through backend preferences, and applies application-matched scrollbars globally.

## Known Work

Run visual smoke checks against the packaged Wails window after large layout changes.
