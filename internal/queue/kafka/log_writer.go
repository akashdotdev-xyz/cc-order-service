package kafka

import (
	"context"
	"log"
)

// LogWriter is a Writer that logs messages instead of sending them to a
// real broker. Useful for local development without a Kafka cluster —
// swap in a real client's writer (e.g. segmentio/kafka-go) once one is
// available.
type LogWriter struct {
	Topic string
}

func (w *LogWriter) WriteMessages(ctx context.Context, msgs ...Message) error {
	for _, m := range msgs {
		log.Printf("[kafka:%s] key=%s value=%s", w.Topic, m.Key, m.Value)
	}
	return nil
}
