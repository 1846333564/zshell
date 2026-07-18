package configstore

import (
	"path/filepath"
	"testing"

	"wiShell/backend/internal/model"
)

func TestLoadMigratesLegacyConfigWhenCurrentMissing(t *testing.T) {
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "wiShell", connectionConfigFile)
	legacyPath := filepath.Join(dir, "zShell", connectionConfigFile)
	legacyStore := Store{path: legacyPath}
	legacyFile := File{
		Version: 1,
		Connections: []model.Connection{
			{ID: "conn-1", Name: "prod", Host: "example.com", Port: 22, Username: "root", AuthMethod: "password", WorkMode: "ops"},
		},
		Preferences: Preferences{UIScale: 1.1, TerminalFontSize: 16, ThemeKey: "nord"},
	}
	if err := legacyStore.saveFile(legacyFile); err != nil {
		t.Fatalf("save legacy config: %v", err)
	}

	store := Store{path: currentPath, legacyPaths: []string{legacyPath}}
	connections, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(connections) != 1 || connections[0].ID != "conn-1" {
		t.Fatalf("Load() connections = %#v, want migrated legacy connection", connections)
	}

	currentFile, exists, err := loadConfigFile(currentPath)
	if err != nil {
		t.Fatalf("load migrated current config: %v", err)
	}
	if !exists {
		t.Fatal("current config was not created")
	}
	if !currentFile.LegacyMigrated {
		t.Fatal("current config did not record legacy migration")
	}
	if currentFile.Preferences.ThemeKey != "nord" {
		t.Fatalf("current preferences theme = %q, want nord", currentFile.Preferences.ThemeKey)
	}
}

func TestLoadBackfillsLegacyConnectionsWhenCurrentHasOnlyPreferences(t *testing.T) {
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "wiShell", connectionConfigFile)
	legacyPath := filepath.Join(dir, "zShell", connectionConfigFile)

	currentStore := Store{path: currentPath}
	if err := currentStore.saveFile(File{
		Version:     1,
		Preferences: Preferences{UIScale: 1.2, TerminalFontSize: 15, ThemeKey: "dracula"},
	}); err != nil {
		t.Fatalf("save current config: %v", err)
	}
	legacyStore := Store{path: legacyPath}
	if err := legacyStore.saveFile(File{
		Version: 1,
		Connections: []model.Connection{
			{ID: "legacy", Name: "legacy", Host: "legacy.example.com", Port: 2222, Username: "ubuntu", AuthMethod: "password", WorkMode: "backend"},
		},
		Preferences: Preferences{UIScale: 0.9, TerminalFontSize: 13, ThemeKey: "tokyo-night"},
	}); err != nil {
		t.Fatalf("save legacy config: %v", err)
	}

	store := Store{path: currentPath, legacyPaths: []string{legacyPath}}
	connections, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(connections) != 1 || connections[0].ID != "legacy" {
		t.Fatalf("Load() connections = %#v, want legacy backfill", connections)
	}

	preferences, err := store.LoadPreferences()
	if err != nil {
		t.Fatalf("LoadPreferences() error = %v", err)
	}
	if preferences.ThemeKey != "dracula" {
		t.Fatalf("preferences theme = %q, want current preference dracula", preferences.ThemeKey)
	}
}

func TestPreferencesEmptyTreatsGPUSettingAsPreference(t *testing.T) {
	enabled := true
	if preferencesEmpty(Preferences{GPUAccelerationEnabled: &enabled}) {
		t.Fatal("GPU acceleration setting must keep current preferences from being replaced")
	}
}
