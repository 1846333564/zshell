package httpapi

import (
	"errors"
	"math"
	"strings"

	"wiShell/backend/internal/configstore"
	"wiShell/backend/internal/model"
)

const defaultThemeKey = "wiShell"

var allowedThemeKeys = map[string]struct{}{
	"wiShell":          {},
	"dracula":          {},
	"nord":             {},
	"tokyo-night":      {},
	"catppuccin-mocha": {},
	"gruvbox-dark":     {},
	"one-dark":         {},
	"solarized-dark":   {},
	"custom":           {},
}

var allowedCustomThemeFields = map[string]struct{}{
	"background":         {},
	"backgroundAlt":      {},
	"backgroundElevated": {},
	"panel":              {},
	"line":               {},
	"primary":            {},
	"primaryAlt":         {},
	"danger":             {},
	"text":               {},
	"muted":              {},
}

func connectionFromRequest(req createConnectionRequest, existing model.Connection) model.Connection {
	authMethod := normalizeAuthMethod(req.AuthMethod)
	workMode := normalizeWorkMode(req.WorkMode)
	password := strings.TrimSpace(req.Password)
	if authMethod == "password" && password == "" {
		password = existing.Password
	}
	if authMethod != "password" {
		password = ""
	}

	id := strings.TrimSpace(existing.ID)
	if id == "" {
		id = strings.TrimSpace(req.ID)
	}

	return model.Connection{
		ID:         id,
		Name:       strings.TrimSpace(req.Name),
		Host:       strings.TrimSpace(req.Host),
		Port:       req.Port,
		Username:   strings.TrimSpace(req.Username),
		Password:   password,
		AuthMethod: authMethod,
		WorkMode:   workMode,
		Hardware:   existing.Hardware,
	}
}

func validateConnectionRequest(req createConnectionRequest, existing model.Connection) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.Host) == "" {
		return errors.New("host is required")
	}
	if req.Port < 1 || req.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("username is required")
	}
	authMethod := normalizeAuthMethod(req.AuthMethod)
	if authMethod != "password" && authMethod != "id_rsa" {
		return errors.New("authMethod must be password or id_rsa")
	}
	if authMethod == "password" && strings.TrimSpace(req.Password) == "" && strings.TrimSpace(existing.Password) == "" {
		return errors.New("password is required")
	}
	workMode := normalizeWorkMode(req.WorkMode)
	if workMode != "frontend" && workMode != "backend" && workMode != "ops" {
		return errors.New("workMode must be frontend, backend or ops")
	}
	return nil
}

func (s *Server) saveConnectionConfigs() error {
	if s.configStore == nil {
		return errors.New("connection config store unavailable")
	}
	if err := s.configStore.Save(s.store.ListFull()); err != nil {
		return err
	}
	return nil
}

func (s *Server) loadUIPreferences() (configstore.Preferences, error) {
	if s.configStore == nil {
		return configstore.Preferences{}, errors.New("connection config store unavailable")
	}
	preferences, err := s.configStore.LoadPreferences()
	if err != nil {
		return configstore.Preferences{}, err
	}
	return normalizeUIPreferences(preferences), nil
}

func (s *Server) saveUIPreferences(preferences configstore.Preferences) error {
	if s.configStore == nil {
		return errors.New("connection config store unavailable")
	}
	return s.configStore.SavePreferences(normalizeUIPreferences(preferences))
}

func (s *Server) GPUAccelerationEnabled() (bool, error) {
	preferences, err := s.loadUIPreferences()
	if err != nil {
		return true, err
	}
	return *preferences.GPUAccelerationEnabled, nil
}

func normalizeUIPreferences(preferences configstore.Preferences) configstore.Preferences {
	preferences.UIScale = normalizeUIScale(preferences.UIScale)
	preferences.TerminalFontSize = normalizeTerminalFontSize(preferences.TerminalFontSize)
	preferences.ThemeKey = normalizeThemeKey(preferences.ThemeKey)
	preferences.CustomTheme = normalizeCustomTheme(preferences.CustomTheme)
	if preferences.GPUAccelerationEnabled == nil {
		preferences.GPUAccelerationEnabled = boolPointer(true)
	}
	return preferences
}

func boolPointer(value bool) *bool {
	return &value
}

func normalizeUIScale(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 1
	}
	value = math.Min(1.35, math.Max(0.82, value))
	return math.Round(value*100) / 100
}

func normalizeTerminalFontSize(value int) int {
	if value <= 0 {
		return 14
	}
	if value < 10 {
		return 10
	}
	if value > 28 {
		return 28
	}
	return value
}

func normalizeThemeKey(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedThemeKeys[key]; ok {
		return key
	}
	return defaultThemeKey
}

func normalizeCustomTheme(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	cleaned := make(map[string]string, len(value))
	for key, color := range value {
		if _, ok := allowedCustomThemeFields[key]; !ok {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(color))
		if !isHexColor(normalized) {
			continue
		}
		cleaned[key] = normalized
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func isHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func normalizeAuthMethod(value string) string {
	authMethod := strings.ToLower(strings.TrimSpace(value))
	if authMethod == "" {
		return "password"
	}
	return authMethod
}

func normalizeWorkMode(value string) string {
	return model.NormalizeWorkMode(value)
}
