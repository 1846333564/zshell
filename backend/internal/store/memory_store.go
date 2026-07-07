package store

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"

	"wiShell/backend/internal/model"
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

func (s *MemoryStore) Put(conn model.Connection) model.Connection {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conn.ID == "" {
		conn.ID = generateID()
	}
	s.connections[conn.ID] = conn
	return conn
}

func (s *MemoryStore) List() []model.ConnectionSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	connections := s.listLocked()
	result := make([]model.ConnectionSummary, 0, len(connections))
	for _, conn := range connections {
		result = append(result, conn.Summary())
	}
	return result
}

func (s *MemoryStore) ListFull() []model.Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listLocked()
}

func (s *MemoryStore) listLocked() []model.Connection {
	result := make([]model.Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		result = append(result, conn)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		if result[i].Host != result[j].Host {
			return result[i].Host < result[j].Host
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func (s *MemoryStore) Get(id string) (model.Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.connections[id]
	return conn, ok
}

func (s *MemoryStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.connections[id]; !ok {
		return false
	}
	delete(s.connections, id)
	return true
}

func generateID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "conn-fallback"
	}
	return "conn-" + hex.EncodeToString(buf)
}
