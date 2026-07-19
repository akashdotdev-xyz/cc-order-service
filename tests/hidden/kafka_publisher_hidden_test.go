package hidden

// Hidden tests for Problem 3 (Add Kafka Events): kafka.Publisher must
// serialize the event correctly and key messages by order ID.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"order-service/internal/queue"
	"order-service/internal/queue/kafka"
)

type fakeKafkaWriter struct {
	messages []kafka.Message
}

func (f *fakeKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	f.messages = append(f.messages, msgs...)
	return nil
}

func TestHidden_KafkaPublisher_KeysMessageByOrderID(t *testing.T) {
	writer := &fakeKafkaWriter{}
	pub := kafka.NewPublisher(writer, "order.created")

	event := queue.OrderCreatedEvent{
		OrderID:     "order-42",
		CustomerID:  "cust-1",
		WarehouseID: "wh-1",
		SellerID:    "seller-1",
		TotalAmount: 1500,
		OccurredAt:  time.Now(),
	}

	err := pub.PublishOrderCreated(context.Background(), event)
	require.NoError(t, err)
	require.Len(t, writer.messages, 1)
	require.Equal(t, "order-42", string(writer.messages[0].Key),
		"messages for the same order should be keyed by orderId so they land on the same partition")

	var decoded queue.OrderCreatedEvent
	require.NoError(t, json.Unmarshal(writer.messages[0].Value, &decoded))
	require.Equal(t, event.OrderID, decoded.OrderID)
	require.Equal(t, event.TotalAmount, decoded.TotalAmount)
}
