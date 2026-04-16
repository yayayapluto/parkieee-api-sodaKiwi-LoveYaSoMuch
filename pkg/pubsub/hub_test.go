// pkg/pubsub/hub_test.go
package pubsub_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
)

func TestHub_SubscribePublish(t *testing.T) {
	hub := pubsub.NewHub()
	ch := hub.Subscribe("gate:abc")
	defer hub.Unsubscribe("gate:abc", ch)

	hub.Publish("gate:abc", `{"type":"open_barrier"}`)

	select {
	case msg := <-ch:
		assert.Equal(t, `{"type":"open_barrier"}`, msg)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestHub_UnsubscribeStopsDelivery(t *testing.T) {
	hub := pubsub.NewHub()
	ch := hub.Subscribe("gate:xyz")
	hub.Unsubscribe("gate:xyz", ch)

	hub.Publish("gate:xyz", "msg")

	select {
	case <-ch:
		t.Fatal("received message after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestHub_MultipleSubscribers(t *testing.T) {
	hub := pubsub.NewHub()
	ch1 := hub.Subscribe("chan:1")
	ch2 := hub.Subscribe("chan:1")
	defer hub.Unsubscribe("chan:1", ch1)
	defer hub.Unsubscribe("chan:1", ch2)

	hub.Publish("chan:1", "broadcast")

	for _, ch := range []chan string{ch1, ch2} {
		select {
		case msg := <-ch:
			assert.Equal(t, "broadcast", msg)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for subscriber")
		}
	}
}
