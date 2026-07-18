package httpapi

import (
	"testing"

	"wiShell/backend/internal/configstore"
)

func TestNormalizeUIPreferencesEnablesGPUByDefault(t *testing.T) {
	preferences := normalizeUIPreferences(configstore.Preferences{})
	if preferences.GPUAccelerationEnabled == nil || !*preferences.GPUAccelerationEnabled {
		t.Fatalf("GPU acceleration default = %v, want enabled", preferences.GPUAccelerationEnabled)
	}
}

func TestNormalizeUIPreferencesPreservesDisabledGPU(t *testing.T) {
	preferences := normalizeUIPreferences(configstore.Preferences{
		GPUAccelerationEnabled: boolPointer(false),
	})
	if preferences.GPUAccelerationEnabled == nil || *preferences.GPUAccelerationEnabled {
		t.Fatalf("GPU acceleration = %v, want disabled", preferences.GPUAccelerationEnabled)
	}
}
