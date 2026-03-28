package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"zshell/backend/internal/model"
)

type MemoryStore struct {
	mu          sync.RWMutex
	connections map[string]model.Connection
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{connections: make(map[string]model.Connection)}
}

func (s *MemoryStore) Add(conn model.Connection) model.Connection {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn.ID = generateID()
	s.connections[conn.ID] = conn
	return conn
}

func (s *MemoryStore) List() []model.ConnectionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.ConnectionSummary, 0, len(s.connections))
	for _, conn := range s.connections {
		result = append(result, conn.Summary())
	}
	return result
}

func (s *MemoryStore) Get(id string) (model.Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.connections[id]
	return conn, ok
}

func generateID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "conn-fallback"
	}
	return "conn-" + hex.EncodeToString(buf)
}
