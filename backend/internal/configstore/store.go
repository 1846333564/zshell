package configstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"zshell/backend/internal/model"
)

type Store struct {
	path string
}

type File struct {
	Version     int                `json:"version"`
	Connections []model.Connection `json:"connections"`
	Preferences Preferences        `json:"preferences"`
}

type Preferences struct {
	UIScale          float64           `json:"uiScale,omitempty"`
	TerminalFontSize int               `json:"terminalFontSize,omitempty"`
	ThemeKey         string            `json:"themeKey,omitempty"`
	CustomTheme      map[string]string `json:"customTheme,omitempty"`
}

func NewDefault() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config dir: %w", err)
	}

	return &Store{path: filepath.Join(dir, "zShell", "connections.dpapi")}, nil
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
	encrypted, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return File{Version: 1}, nil
		}
		return File{}, fmt.Errorf("read connection config: %w", err)
	}

	plain, err := decrypt(encrypted)
	if err != nil {
		return File{}, fmt.Errorf("decrypt connection config: %w", err)
	}

	var file File
	if err := json.Unmarshal(plain, &file); err != nil {
		return File{}, fmt.Errorf("decode connection config: %w", err)
	}
	if file.Version == 0 {
		file.Version = 1
	}

	return file, nil
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
