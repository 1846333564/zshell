package configstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"wiShell/backend/internal/model"
)

const connectionConfigFile = "connections.dpapi"

type Store struct {
	path        string
	legacyPaths []string
}

type File struct {
	Version        int                `json:"version"`
	Connections    []model.Connection `json:"connections"`
	Preferences    Preferences        `json:"preferences"`
	LegacyMigrated bool               `json:"legacyMigrated,omitempty"`
}

type Preferences struct {
	UIScale                float64           `json:"uiScale,omitempty"`
	TerminalFontSize       int               `json:"terminalFontSize,omitempty"`
	ThemeKey               string            `json:"themeKey,omitempty"`
	CustomTheme            map[string]string `json:"customTheme,omitempty"`
	GPUAccelerationEnabled *bool             `json:"gpuAccelerationEnabled,omitempty"`
}

func NewDefault() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config dir: %w", err)
	}

	return &Store{
		path: filepath.Join(dir, "wiShell", connectionConfigFile),
		legacyPaths: []string{
			filepath.Join(dir, "zShell", connectionConfigFile),
			filepath.Join(dir, "zshell", connectionConfigFile),
		},
	}, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() ([]model.Connection, error) {
	file, err := s.loadFile()
	if err != nil {
		return nil, err
	}

	return file.Connections, nil
}

func (s *Store) LoadPreferences() (Preferences, error) {
	file, err := s.loadFile()
	if err != nil {
		return Preferences{}, err
	}

	return file.Preferences, nil
}

func (s *Store) Save(connections []model.Connection) error {
	file, err := s.loadFile()
	if err != nil {
		return err
	}
	file.Connections = connections
	return s.saveFile(file)
}

func (s *Store) SavePreferences(preferences Preferences) error {
	file, err := s.loadFile()
	if err != nil {
		return err
	}
	file.Preferences = preferences
	return s.saveFile(file)
}

func (s *Store) loadFile() (File, error) {
	file, exists, err := loadConfigFile(s.path)
	if err != nil {
		return File{}, err
	}
	if !exists {
		return s.migrateLegacyFile(File{Version: 1})
	}
	if !file.LegacyMigrated && len(file.Connections) == 0 {
		return s.migrateLegacyFile(file)
	}

	return file, nil
}

func loadConfigFile(path string) (File, bool, error) {
	encrypted, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{}, false, nil
		}
		return File{}, false, fmt.Errorf("read connection config: %w", err)
	}

	plain, err := decrypt(encrypted)
	if err != nil {
		return File{}, false, fmt.Errorf("decrypt connection config: %w", err)
	}

	var file File
	if err := json.Unmarshal(plain, &file); err != nil {
		return File{}, false, fmt.Errorf("decode connection config: %w", err)
	}
	if file.Version == 0 {
		file.Version = 1
	}

	return file, true, nil
}

func (s *Store) migrateLegacyFile(current File) (File, error) {
	for _, legacyPath := range s.legacyPaths {
		legacy, exists, err := loadConfigFile(legacyPath)
		if err != nil {
			return File{}, fmt.Errorf("load legacy connection config: %w", err)
		}
		if !exists {
			continue
		}

		if len(current.Connections) == 0 && len(legacy.Connections) > 0 {
			current.Connections = legacy.Connections
		}
		if preferencesEmpty(current.Preferences) && !preferencesEmpty(legacy.Preferences) {
			current.Preferences = legacy.Preferences
		}
		current.LegacyMigrated = true
		if current.Version == 0 {
			current.Version = 1
		}
		if err := s.saveFile(current); err != nil {
			return File{}, fmt.Errorf("migrate legacy connection config: %w", err)
		}
		return current, nil
	}

	return current, nil
}

func preferencesEmpty(preferences Preferences) bool {
	return preferences.UIScale == 0 &&
		preferences.TerminalFontSize == 0 &&
		preferences.ThemeKey == "" &&
		len(preferences.CustomTheme) == 0 &&
		preferences.GPUAccelerationEnabled == nil
}

func (s *Store) saveFile(file File) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if file.Version == 0 {
		file.Version = 1
	}

	payload, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode connection config: %w", err)
	}

	encrypted, err := encrypt(payload)
	if err != nil {
		return fmt.Errorf("encrypt connection config: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, encrypted, 0o600); err != nil {
		return fmt.Errorf("write temp connection config: %w", err)
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("remove old connection config: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace connection config: %w", err)
	}

	return nil
}
