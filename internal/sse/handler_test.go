package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
)

func TestGateChannel_UsesPrefixedFormat(t *testing.T) {
	assert.Equal(t, "gate:abc123", GateChannel("abc123"))
}

func TestRegisterRoutes_RegistersSSEPaths(t *testing.T) {
	app := fiber.New()
	h := NewHandler(pubsub.NewHub())

	router := app.Group("/api/v1")
	RegisterRoutes(router, h, nil, "secret")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sse/unknown", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
