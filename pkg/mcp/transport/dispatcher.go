package transport

import (
	"context"
	"errors"
	"sync"
	"time"

	"openmanus-go/pkg/mcp"
)

// Dispatcher correlates JSON-RPC responses (by id) coming from SSE with
// the goroutines waiting for them.
type Dispatcher struct {
	mu      sync.Mutex
	waiters map[string]chan *mcp.Message
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{waiters: make(map[string]chan *mcp.Message)}
}

var GlobalDispatcher = NewDispatcher()

// Register creates a channel for the given id and returns it.
func (d *Dispatcher) Register(id string) chan *mcp.Message {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan *mcp.Message, 1)
	d.waiters[id] = ch
	return ch
}

// Deliver attempts to deliver the message to a waiting goroutine by id.
func (d *Dispatcher) Deliver(msg *mcp.Message) {
	if msg == nil || msg.ID == nil {
		return
	}
	id := *msg.ID
	d.mu.Lock()
	ch, ok := d.waiters[id]
	if ok {
		delete(d.waiters, id)
	}
	d.mu.Unlock()
	if ok {
		ch <- msg
		close(ch)
	}
}

// Wait waits for a response with given id until ctx done or timeout.
func (d *Dispatcher) Wait(ctx context.Context, id string, timeout time.Duration) (*mcp.Message, error) {
	ch := d.Register(id)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.New("mcp wait timed out")
	case msg := <-ch:
		return msg, nil
	}
}
