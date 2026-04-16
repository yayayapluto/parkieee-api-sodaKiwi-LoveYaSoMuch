package sse

import (
	"bufio"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
)

const cashierChannel = "cashier:events"

type Handler struct {
	hub *pubsub.Hub
}

func NewHandler(hub *pubsub.Hub) *Handler {
	return &Handler{hub: hub}
}

func GateChannel(gateID string) string {
	return "gate:" + gateID
}

func (h *Handler) GateEvents(c *fiber.Ctx) error {
	gateID := c.Params("id")
	channel := GateChannel(gateID)
	ch := h.hub.Subscribe(channel)
	defer h.hub.Unsubscribe(channel, ch)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case <-c.Context().Done():
				return
			case msg := <-ch:
				_, _ = fmt.Fprintf(w, "data: %s\n\n", NormalizeEventMessage(msg))
				_ = w.Flush()
			}
		}
	})

	return nil
}

func (h *Handler) CashierEvents(c *fiber.Ctx) error {
	ch := h.hub.Subscribe(cashierChannel)
	defer h.hub.Unsubscribe(cashierChannel, ch)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case <-c.Context().Done():
				return
			case msg := <-ch:
				_, _ = fmt.Fprintf(w, "data: %s\n\n", NormalizeEventMessage(msg))
				_ = w.Flush()
			}
		}
	})

	return nil
}
