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

Terminal and monitor panels are integrated. `ConnectionForm.vue` supports create/edit modes. `TerminalPanel.vue` handles terminal-focused font sizing, terminal clipboard shortcuts, and a terminal context menu for copy/paste/clear/reconnect. `FileManager.vue` uses a top path input, a collapsible path navigator with fixed `/` root visibility, resolved home paths such as `/root`, basename-only tree labels, color-only opened markers, right-side centered fold buttons, full context-menu file actions, drag/drop upload, picker upload, compact upload progress with total/per-file/speed details, and resizable file columns.

## Known Work

Keep file actions in context menus rather than returning fixed toolbar buttons. Verify context-menu behavior against real files and folders when a server is available.
