package model

import "fmt"

type Connection struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

type ConnectionSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func (c Connection) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c Connection) Summary() ConnectionSummary {
	return ConnectionSummary{
		ID:       c.ID,
		Name:     c.Name,
		Host:     c.Host,
		Port:     c.Port,
		Username: c.Username,
	}
}
