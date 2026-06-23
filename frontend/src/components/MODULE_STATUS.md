# Frontend Components Module Status

## Scope

Vue components for connection forms, monitor panel, terminal tabs/panels, and file manager.

## Important Files

- `ConnectionForm.vue`
- `MonitorPanel.vue`
- `TerminalTabs.vue`
- `TerminalPanel.vue`
- `FileManager.vue`

## Current State

Terminal and monitor panels are integrated. `ConnectionForm.vue` supports create/edit modes. `FileManager.vue` uses a top path input, a collapsible path navigator with fixed `/` root visibility, resolved home paths such as `/root`, folder-first listing, and a refresh-only context menu.

## Known Work

Reintroduce upload/download-selection/copy/move only through a deliberate context-menu design if those controls are needed again.
