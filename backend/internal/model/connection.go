package model

import (
	"fmt"
	"strings"
)

type Connection struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	AuthMethod string `json:"authMethod"`
	WorkMode   string `json:"workMode"`
}

type ConnectionSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	AuthMethod string `json:"authMethod"`
	WorkMode   string `json:"workMode"`
}

func (c Connection) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c Connection) Summary() ConnectionSummary {
	return ConnectionSummary{
		ID:         c.ID,
		Name:       c.Name,
		Host:       c.Host,
		Port:       c.Port,
		Username:   c.Username,
		AuthMethod: c.AuthMethod,
		WorkMode:   NormalizeWorkMode(c.WorkMode),
	}
}

func NormalizeWorkMode(value string) string {
	workMode := strings.ToLower(strings.TrimSpace(value))
	if workMode == "" {
		return "ops"
	}
	switch workMode {
	case "frontend", "front":
		return "frontend"
	case "backend", "back":
		return "backend"
	case "ops", "operation", "operations", "devops":
		return "ops"
	default:
		return workMode
	}
}
