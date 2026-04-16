// pkg/event/bus.go
package event

import "log"

type EventType string

type Event struct {
	Type    EventType
	Payload any
}

type Bus struct {
	ch          chan Event
	subscribers []chan Event
}

func NewBus(bufferSize int) *Bus {
	b := &Bus{ch: make(chan Event, bufferSize)}
	go b.dispatch()
	return b
}

func (b *Bus) Publish(e Event) {
	select {
	case b.ch <- e:
	default:
		log.Printf("event bus full, dropping event type=%s", e.Type)
	}
}

func (b *Bus) Subscribe() <-chan Event {
	ch := make(chan Event, 100)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

func (b *Bus) dispatch() {
	for e := range b.ch {
		for _, sub := range b.subscribers {
			select {
			case sub <- e:
			default:
				log.Printf("subscriber buffer full, dropping event type=%s", e.Type)
			}
		}
	}
}
