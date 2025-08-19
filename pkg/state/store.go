package state

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store 定义状态存储接口
type Store interface {
	Save(trace *Trace) error
	Load(id string) (*Trace, error)
	List() ([]string, error)
	Delete(id string) error
}

// FileStore 基于文件系统的存储实现
type FileStore struct {
	basePath string
}

// NewFileStore 创建文件存储实例
func NewFileStore(basePath string) *FileStore {
	return &FileStore{
		basePath: basePath,
	}
}

// Save 保存轨迹到文件
func (s *FileStore) Save(trace *Trace) error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 使用时间戳作为文件名
	filename := fmt.Sprintf("trace_%d.json", time.Now().Unix())
	filepath := filepath.Join(s.basePath, filename)

	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trace: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load 从文件加载轨迹
func (s *FileStore) Load(id string) (*Trace, error) {
	filePath := filepath.Join(s.basePath, id)
	if !strings.HasSuffix(filePath, ".json") {
		filePath += ".json"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var trace Trace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace: %w", err)
	}

	return &trace, nil
}

// List 列出所有轨迹文件
func (s *FileStore) List() ([]string, error) {
	var files []string

	err := filepath.WalkDir(s.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && filepath.Ext(path) == ".json" {
			relPath, err := filepath.Rel(s.basePath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// Delete 删除轨迹文件
func (s *FileStore) Delete(id string) error {
	filePath := filepath.Join(s.basePath, id)
	if !strings.HasSuffix(filePath, ".json") {
		filePath += ".json"
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// MemoryStore 基于内存的存储实现（用于测试）
type MemoryStore struct {
	traces map[string]*Trace
}

// NewMemoryStore 创建内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		traces: make(map[string]*Trace),
	}
}

// Save 保存轨迹到内存
func (ms *MemoryStore) Save(trace *Trace) error {
	id := fmt.Sprintf("trace_%d", time.Now().Unix())

	// 深拷贝轨迹
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}

	var copy Trace
	if err := json.Unmarshal(data, &copy); err != nil {
		return err
	}

	ms.traces[id] = &copy
	return nil
}

// Load 从内存加载轨迹
func (ms *MemoryStore) Load(id string) (*Trace, error) {
	trace, ok := ms.traces[id]
	if !ok {
		return nil, fmt.Errorf("trace not found: %s", id)
	}

	// 深拷贝轨迹
	data, err := json.Marshal(trace)
	if err != nil {
		return nil, err
	}

	var copy Trace
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}

	return &copy, nil
}

// List 列出所有轨迹ID
func (ms *MemoryStore) List() ([]string, error) {
	var ids []string
	for id := range ms.traces {
		ids = append(ids, id)
	}
	return ids, nil
}

// Delete 从内存删除轨迹
func (ms *MemoryStore) Delete(id string) error {
	if _, ok := ms.traces[id]; !ok {
		return fmt.Errorf("trace not found: %s", id)
	}
	delete(ms.traces, id)
	return nil
}
