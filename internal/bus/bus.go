
package bus

import "sync"

type Event struct {
	Topic string
	Data  map[string]any
}

type Subscriber chan Event

type Bus struct {
	mu   sync.RWMutex
	subs map[string][]Subscriber
}

func New() *Bus { return &Bus{subs: map[string][]Subscriber{}} }

func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[e.Topic] {
		select { case ch <- e: default: }
	}
}

func (b *Bus) Subscribe(topic string) Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(Subscriber, 64)
	b.subs[topic] = append(b.subs[topic], ch)
	return ch
}
