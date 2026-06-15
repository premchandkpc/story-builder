package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/event"
	"github.com/segmentio/kafka-go"
)

type Bus struct {
	writer   *kafka.Writer
	reader   *kafka.Reader
	brokers  []string
	topic    string
	groupID  string
	mu       sync.RWMutex
	handlers map[event.EventType][]func(*event.Event) error
	stop     chan struct{}
}

func NewBus(brokers []string, topic, groupID string) *Bus {
	return &Bus{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
		},
		brokers:  brokers,
		topic:    topic,
		groupID:  groupID,
		handlers: make(map[event.EventType][]func(*event.Event) error),
		stop:     make(chan struct{}),
	}
}

func (b *Bus) Publish(evt *event.Event) error {
	if evt.ID == uuid.Nil {
		evt.ID = uuid.New()
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("kafka marshal: %w", err)
	}
	return b.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(evt.AggregateID.String()),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(evt.Type)},
			{Key: "aggregate_id", Value: []byte(evt.AggregateID.String())},
		},
	})
}

func (b *Bus) Subscribe(evtType event.EventType, handler func(*event.Event) error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[evtType] = append(b.handlers[evtType], handler)
}

func (b *Bus) Start() {
	b.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  b.brokers,
		Topic:    b.topic,
		GroupID:  b.groupID,
		MinBytes: 10,
		MaxBytes: 10e6,
	})
	go b.consume()
}

func (b *Bus) Stop() {
	close(b.stop)
	b.writer.Close()
	if b.reader != nil {
		b.reader.Close()
	}
}

func (b *Bus) consume() {
	for {
		select {
		case <-b.stop:
			return
		default:
		}

		msg, err := b.reader.ReadMessage(context.Background())
		if err != nil {
			slog.Error("kafka read", "error", err)
			continue
		}

		var evt event.Event
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			slog.Error("kafka unmarshal", "error", err)
			continue
		}

		b.mu.RLock()
		handlers := b.handlers[evt.Type]
		b.mu.RUnlock()

		for _, h := range handlers {
			if err := h(&evt); err != nil {
				slog.Error("event handler", "type", evt.Type, "error", err)
			}
		}
	}
}
