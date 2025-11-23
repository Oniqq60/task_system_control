package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// TaskEvent представляет событие изменения задачи для Kafka
type TaskEvent struct {
	TaskID    string    `json:"taskId"`
	UserID    string    `json:"userId"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type KafkaProducer interface {
	SendTaskEvent(ctx context.Context, event TaskEvent) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
	topic  string
}

func NewKafkaProducer(brokers []string, topic string) KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &kafkaProducer{
		writer: writer,
		topic:  topic,
	}
}

// SendTaskEvent отправляет событие задачи в Kafka
func (p *kafkaProducer) SendTaskEvent(ctx context.Context, event TaskEvent) error {

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	message := kafka.Message{
		Key:   []byte(event.TaskID),
		Value: eventJSON,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, message)
}

// Close закрывает соединение с Kafka
func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}
