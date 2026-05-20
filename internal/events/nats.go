package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSClient wraps NATS JetStream connection
type NATSClient struct {
	Conn      *nats.Conn
	JetStream jetstream.JetStream
}

// NewNATSClient connects to NATS and initializes JetStream
func NewNATSClient(url string) (*NATSClient, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			slog.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("nats reconnected", "url", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats: failed to connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: failed to create jetstream context: %w", err)
	}

	slog.Info("nats connected", "url", nc.ConnectedUrl())

	return &NATSClient{
		Conn:      nc,
		JetStream: js,
	}, nil
}

// EnsureStreams creates all required streams if they don't exist
func (c *NATSClient) EnsureStreams(ctx context.Context) error {
	for name, subjects := range StreamConfig {
		_, err := c.JetStream.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:       name,
			Subjects:   subjects,
			Retention:  jetstream.WorkQueuePolicy,
			MaxAge:     14 * 24 * time.Hour, // 14 days retention
			Storage:    jetstream.FileStorage,
			Replicas:   1,
			Discard:    jetstream.DiscardOld,
			MaxMsgs:    -1,
			MaxBytes:   -1,
			Duplicates: 5 * time.Minute, // Dedup window
		})
		if err != nil {
			return fmt.Errorf("nats: failed to create stream %s: %w", name, err)
		}
		slog.Info("nats stream ready", "stream", name, "subjects", subjects)
	}
	return nil
}

// Publish publishes an event to NATS JetStream
func (c *NATSClient) Publish(ctx context.Context, event *Event) error {
	data, err := event.Marshal()
	if err != nil {
		return fmt.Errorf("nats: failed to marshal event: %w", err)
	}

	opts := []jetstream.PublishOpt{}
	if event.IdempotencyKey != nil {
		opts = append(opts, jetstream.WithMsgID(*event.IdempotencyKey))
	} else {
		opts = append(opts, jetstream.WithMsgID(event.ID))
	}

	_, err = c.JetStream.Publish(ctx, event.Subject(), data, opts...)
	if err != nil {
		return fmt.Errorf("nats: failed to publish event %s: %w", event.Type, err)
	}

	return nil
}

// Subscribe creates a durable consumer for a stream
func (c *NATSClient) Subscribe(ctx context.Context, stream, consumerName string, handler func(jetstream.Msg)) (jetstream.ConsumeContext, error) {
	consumer, err := c.JetStream.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5, // Max 5 retries before giving up
		FilterSubject: "", // All subjects in stream
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("nats: failed to create consumer %s: %w", consumerName, err)
	}

	cc, err := consumer.Consume(handler)
	if err != nil {
		return nil, fmt.Errorf("nats: failed to start consuming %s: %w", consumerName, err)
	}

	slog.Info("nats consumer started", "stream", stream, "consumer", consumerName)
	return cc, nil
}

// Close closes the NATS connection
func (c *NATSClient) Close() {
	if c.Conn != nil {
		c.Conn.Drain()
		slog.Info("nats connection closed")
	}
}

// HealthCheck checks if NATS is connected
func (c *NATSClient) HealthCheck() error {
	if !c.Conn.IsConnected() {
		return fmt.Errorf("nats: not connected")
	}
	return nil
}
