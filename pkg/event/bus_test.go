// pkg/event/bus_test.go
package event_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/pkg/event"
)

func TestBus_PublishSubscribe(t *testing.T) {
	bus := event.NewBus(10)
	ch := bus.Subscribe()

	bus.Publish(event.Event{Type: "test_event", Payload: "hello"})

	select {
	case ev := <-ch:
		assert.Equal(t, event.EventType("test_event"), ev.Type)
		assert.Equal(t, "hello", ev.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestBus_FullBufferDropsSilently(t *testing.T) {
	bus := event.NewBus(1)
	_ = bus.Subscribe()
	// Fill buffer
	bus.Publish(event.Event{Type: "e1", Payload: nil})
	// Should not block
	done := make(chan struct{})
	go func() {
		bus.Publish(event.Event{Type: "e2", Payload: nil})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on full buffer")
	}
}
