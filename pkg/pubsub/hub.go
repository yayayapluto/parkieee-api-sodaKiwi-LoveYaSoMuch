// pkg/pubsub/hub.go
package pubsub

import "sync"

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan string
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string][]chan string)}
}

func (h *Hub) Subscribe(channel string) chan string {
	ch := make(chan string, 50)
	h.mu.Lock()
	h.subscribers[channel] = append(h.subscribers[channel], ch)
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(channel string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.subscribers[channel]
	for i, s := range subs {
		if s == ch {
			h.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

func (h *Hub) Publish(channel, msg string) {
	h.mu.RLock()
	subs := make([]chan string, len(h.subscribers[channel]))
	copy(subs, h.subscribers[channel])
	h.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// subscriber buffer full — drop
		}
	}
}
