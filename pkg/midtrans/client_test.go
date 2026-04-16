// pkg/midtrans/client_test.go
package midtrans_test

import (
	"crypto/sha512"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/pkg/midtrans"
)

func TestValidateCallback(t *testing.T) {
	serverKey := "test-server-key"
	orderID := "order-123"
	statusCode := "200"
	grossAmount := "5000.00"

	raw := orderID + statusCode + grossAmount + serverKey
	h := sha512.Sum512([]byte(raw))
	sig := fmt.Sprintf("%x", h)

	client := midtrans.NewClient(serverKey, "client-key", "sandbox")
	assert.True(t, client.ValidateCallback(orderID, statusCode, grossAmount, sig))
	assert.False(t, client.ValidateCallback(orderID, statusCode, grossAmount, "bad-sig"))
}
