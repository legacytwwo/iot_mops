package rabbit

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	mathrand "math/rand"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"rule_engine/internal/config"
)

type Client struct {
	conn   *amqp.Connection
	cfg    config.RabbitConfig
	logger *zap.Logger
}

func New(conn *amqp.Connection, cfg config.RabbitConfig, logger *zap.Logger) *Client {
	seed := make([]byte, 8)
	_, _ = rand.Read(seed)
	mathrand.Seed(int64(binaryToUint64(seed)))

	return &Client{
		conn:   conn,
		cfg:    cfg,
		logger: logger,
	}
}

func (c *Client) SetupTopology() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(c.cfg.Exchange, c.cfg.ExchangeType, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(c.cfg.RetryExchange, c.cfg.RetryExchangeType, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare retry exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(c.cfg.DLXExchange, c.cfg.DLXExchangeType, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx exchange: %w", err)
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange":    c.cfg.DLXExchange,
		"x-dead-letter-routing-key": c.cfg.DLQRoutingKey,
	}
	if _, err := ch.QueueDeclare(c.cfg.Queue, true, false, false, false, mainArgs); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}
	if err := ch.QueueBind(c.cfg.Queue, c.cfg.RoutingKey, c.cfg.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind main queue: %w", err)
	}

	retryArgs := amqp.Table{
		"x-dead-letter-exchange":    c.cfg.Exchange,
		"x-dead-letter-routing-key": c.cfg.RoutingKey,
	}
	if _, err := ch.QueueDeclare(c.cfg.RetryQueue, true, false, false, false, retryArgs); err != nil {
		return fmt.Errorf("declare retry queue: %w", err)
	}
	if err := ch.QueueBind(c.cfg.RetryQueue, c.cfg.RetryRoutingKey, c.cfg.RetryExchange, false, nil); err != nil {
		return fmt.Errorf("bind retry queue: %w", err)
	}

	if _, err := ch.QueueDeclare(c.cfg.DLQQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq queue: %w", err)
	}
	if err := ch.QueueBind(c.cfg.DLQQueue, c.cfg.DLQRoutingKey, c.cfg.DLXExchange, false, nil); err != nil {
		return fmt.Errorf("bind dlq queue: %w", err)
	}

	return nil
}

func (c *Client) Consume(ctx context.Context) (<-chan amqp.Delivery, func() error, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("channel: %w", err)
	}

	if err := ch.Qos(c.cfg.Prefetch, 0, false); err != nil {
		_ = ch.Close()
		return nil, nil, fmt.Errorf("qos: %w", err)
	}

	deliveries, err := ch.Consume(
		c.cfg.Queue,
		c.cfg.ConsumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		return nil, nil, fmt.Errorf("consume: %w", err)
	}

	cleanup := func() error {
		_ = ch.Cancel(c.cfg.ConsumerTag, false)
		return ch.Close()
	}

	go func() {
		<-ctx.Done()
		_ = ch.Cancel(c.cfg.ConsumerTag, false)
	}()

	return deliveries, cleanup, nil
}

func (c *Client) PublishRetry(ctx context.Context, msg amqp.Delivery, attempt int) error {
	delay := NextRetryDelay(c.cfg.RetryBaseDelay, c.cfg.RetryMaxDelay, attempt, c.cfg.RetryJitter)
	headers := copyHeaders(msg.Headers)
	headers["x-retry-count"] = attempt

	return c.publish(ctx, c.cfg.RetryExchange, c.cfg.RetryRoutingKey, msg.Body, headers, delay)
}

func (c *Client) PublishDLQ(ctx context.Context, msg amqp.Delivery, reason string) error {
	headers := copyHeaders(msg.Headers)
	headers["x-error-reason"] = reason

	return c.publish(ctx, c.cfg.DLXExchange, c.cfg.DLQRoutingKey, msg.Body, headers, 0)
}

func (c *Client) publish(ctx context.Context, exchange, routingKey string, body []byte, headers amqp.Table, delay time.Duration) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}
	defer ch.Close()

	pub := amqp.Publishing{
		Headers:      headers,
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}
	if delay > 0 {
		pub.Expiration = strconv.FormatInt(delay.Milliseconds(), 10)
	}

	if err := ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

func RetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	if v, ok := headers["x-retry-count"]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int8:
			return int(val)
		case int16:
			return int(val)
		case int32:
			return int(val)
		case int64:
			return int(val)
		case float32:
			return int(val)
		case float64:
			return int(val)
		case string:
			if n, err := strconv.Atoi(val); err == nil {
				return n
			}
		case json.Number:
			if n, err := val.Int64(); err == nil {
				return int(n)
			}
		}
	}
	return 0
}

func NextRetryDelay(base, max time.Duration, attempt int, jitter float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}

	d := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if d > max {
		d = max
	}

	if d <= 0 {
		return base
	}

	if jitter == 0 {
		return d
	}

	factor := 1 + (mathrand.Float64()*2-1)*jitter
	return time.Duration(float64(d) * factor)
}

func copyHeaders(src amqp.Table) amqp.Table {
	dst := amqp.Table{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func binaryToUint64(b []byte) uint64 {
	var out uint64
	for i := 0; i < len(b) && i < 8; i++ {
		out = (out << 8) | uint64(b[i])
	}
	return out
}
