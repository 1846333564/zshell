package httpapi

import (
	"wiShell/backend/internal/sftpsvc"
)

type createConnectionRequest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	AuthMethod string `json:"authMethod"`
	WorkMode   string `json:"workMode"`
}

type idRequest struct {
	ConnectionID string `json:"connectionId"`
}

type execRequest struct {
	ConnectionID string `json:"connectionId"`
	Command      string `json:"command"`
}

type sftpListRequest struct {
	ConnectionID string `json:"connectionId"`
	Path         string `json:"path"`
}

type sftpFileReadRequest struct {
	ConnectionID string `json:"connectionId"`
	Path         string `json:"path"`
}

type sftpFileWriteRequest struct {
	ConnectionID string `json:"connectionId"`
	Path         string `json:"path"`
	Content      string `json:"content"`
}

type sftpTransferRequest struct {
	SourceConnectionID string                 `json:"sourceConnectionId"`
	TargetConnectionID string                 `json:"targetConnectionId"`
	TargetPath         string                 `json:"targetPath"`
	Action             string                 `json:"action"`
	Items              []sftpsvc.TransferItem `json:"items"`
}

type sftpDeleteRequest struct {
	ConnectionID string                 `json:"connectionId"`
	Items        []sftpsvc.TransferItem `json:"items"`
}

type monitorSnapshotRequest struct {
	ConnectionID string `json:"connectionId"`
	ProcessSort  string `json:"processSort"`
}

type uiPreferencesRequest struct {
	UIScale                *float64          `json:"uiScale"`
	TerminalFontSize       *int              `json:"terminalFontSize"`
	ThemeKey               *string           `json:"themeKey"`
	CustomTheme            map[string]string `json:"customTheme"`
	GPUAccelerationEnabled *bool             `json:"gpuAccelerationEnabled"`
}
