package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"openmanus-go/pkg/logger"
)

// MemoryEntry 带元数据的记忆条目
type MemoryEntry struct {
	Key        string    `json:"key"`
	Value      any       `json:"value"`
	Category   string    `json:"category,omitempty"`
	Importance float64   `json:"importance"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	AccessedAt time.Time `json:"accessed_at"`
	TTL        time.Duration `json:"-"`
	ExpiresAt  *time.Time    `json:"expires_at,omitempty"`
}

// IsExpired 判断条目是否已过期
func (e *MemoryEntry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// MemoryStore 记忆存储接口，可对接不同后端
type MemoryStore interface {
	Get(key string) (*MemoryEntry, bool)
	Set(entry *MemoryEntry)
	Delete(key string)
	List() []*MemoryEntry
	Flush() error
}

// --- InMemoryStore: 内存存储（用于短期记忆） ---

type InMemoryStore struct {
	data map[string]*MemoryEntry
	mu   sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]*MemoryEntry),
	}
}

func (s *InMemoryStore) Get(key string) (*MemoryEntry, bool) {
	s.mu.RLock()
	entry, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if entry.IsExpired() {
		s.Delete(key)
		return nil, false
	}

	s.mu.Lock()
	entry.AccessedAt = time.Now()
	s.mu.Unlock()

	return entry, true
}

func (s *InMemoryStore) Set(entry *MemoryEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[entry.Key] = entry
}

func (s *InMemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *InMemoryStore) List() []*MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*MemoryEntry, 0, len(s.data))
	for _, entry := range s.data {
		if !entry.IsExpired() {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Flush 内存存储无需持久化
func (s *InMemoryStore) Flush() error {
	return nil
}

// CleanExpired 清理过期条目
func (s *InMemoryStore) CleanExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for key, entry := range s.data {
		if entry.IsExpired() {
			delete(s.data, key)
			count++
		}
	}
	return count
}

// --- FileStore: 文件持久化存储（用于长期记忆） ---

type FileStore struct {
	path string
	data map[string]*MemoryEntry
	mu   sync.RWMutex
}

func NewFileStore(path string) (*FileStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	store := &FileStore{
		path: path,
		data: make(map[string]*MemoryEntry),
	}

	if err := store.load(); err != nil {
		logger.Warnw("memory.file_store.load_failed", "path", path, "error", err)
	}

	return store, nil
}

func (s *FileStore) Get(key string) (*MemoryEntry, bool) {
	s.mu.RLock()
	entry, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return nil, false
	}

	s.mu.Lock()
	entry.AccessedAt = time.Now()
	s.mu.Unlock()

	return entry, true
}

func (s *FileStore) Set(entry *MemoryEntry) {
	s.mu.Lock()
	s.data[entry.Key] = entry
	s.mu.Unlock()

	if err := s.persist(); err != nil {
		logger.Warnw("memory.file_store.persist_failed", "error", err)
	}
}

func (s *FileStore) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()

	if err := s.persist(); err != nil {
		logger.Warnw("memory.file_store.persist_failed", "error", err)
	}
}

func (s *FileStore) List() []*MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]*MemoryEntry, 0, len(s.data))
	for _, entry := range s.data {
		entries = append(entries, entry)
	}
	return entries
}

// Flush 将数据持久化到文件
func (s *FileStore) Flush() error {
	return s.persist()
}

func (s *FileStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var entries map[string]*MemoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse memory file: %w", err)
	}

	s.mu.Lock()
	s.data = entries
	s.mu.Unlock()

	logger.Infow("memory.file_store.loaded", "path", s.path, "entries", len(entries))
	return nil
}

func (s *FileStore) persist() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	return nil
}

// NewMemoryEntry 创建记忆条目的便捷方法
func NewMemoryEntry(key string, value any, category string, importance float64) *MemoryEntry {
	now := time.Now()
	return &MemoryEntry{
		Key:        key,
		Value:      value,
		Category:   category,
		Importance: importance,
		CreatedAt:  now,
		UpdatedAt:  now,
		AccessedAt: now,
	}
}

// NewMemoryEntryWithTTL 创建带 TTL 的记忆条目
func NewMemoryEntryWithTTL(key string, value any, category string, importance float64, ttl time.Duration) *MemoryEntry {
	entry := NewMemoryEntry(key, value, category, importance)
	entry.TTL = ttl
	expiresAt := time.Now().Add(ttl)
	entry.ExpiresAt = &expiresAt
	return entry
}
