package notification

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// Consumer читает сообщения из Kafka
type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
	Close() error
}

type EventHandler interface {
	HandleEvent(ctx context.Context, event TaskEvent) error
}

type kafkaConsumer struct {
	reader  *kafka.Reader
	handler EventHandler
	topic   string
	groupID string
}

func NewKafkaConsumer(brokers []string, topic, groupID string, handler EventHandler) Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	return &kafkaConsumer{
		reader:  reader,
		handler: handler,
		topic:   topic,
		groupID: groupID,
	}
}

// Start читает сообщения в цикле до отмены контекста
// TODO: обработка ошибок, json.Unmarshal, graceful shutdown
func (c *kafkaConsumer) Start(ctx context.Context) error {
	log.Printf("Kafka consumer started (topic=%s, group=%s)", c.topic, c.groupID)

	for {
		select {
		case <-ctx.Done():
			log.Println("Kafka consumer context cancelled")
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("read message error: %v", err)
				continue
			}

			var event TaskEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("unmarshal task event error: %v", err)
				continue
			}

			if err := c.handler.HandleEvent(ctx, event); err != nil {
				log.Printf("handle event error: %v", err)
			}
		}
	}
}

func (c *kafkaConsumer) Stop() error {
	return nil
}

func (c *kafkaConsumer) Close() error {
	return c.reader.Close()
}
