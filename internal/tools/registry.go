
package tools

import (
	"context"
	"fmt"
	"sync"
)

type Input = map[string]any
type Output = map[string]any

type Schema struct {
	Name   string            `json:"name"`
	Desc   string            `json:"desc"`
	Inputs map[string]string `json:"inputs"`
}

type Tool interface {
	Name() string
	Desc() string
	Schema() Schema
	Run(ctx context.Context, in Input) (Output, error)
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry { return &Registry{tools: make(map[string]Tool)} }

func (r *Registry) Register(t Tool) { r.mu.Lock(); defer r.mu.Unlock(); r.tools[t.Name()] = t }

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock(); defer r.mu.RUnlock()
	t, ok := r.tools[name]; return t, ok
}

func (r *Registry) List() []Tool {
	r.mu.RLock(); defer r.mu.RUnlock()
	res := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools { res = append(res, t) }
	return res
}

func (r *Registry) Schemas() []Schema {
	ts := r.List()
	res := make([]Schema, 0, len(ts))
	for _, t := range ts { res = append(res, t.Schema()) }
	return res
}

func (r *Registry) MustGet(name string) Tool {
	if t, ok := r.Get(name); ok { return t }
	panic(fmt.Sprintf("tool not found: %s", name))
}
