package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

// Client holds the RabbitMQ connection and channel.
type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	// Add mutex for channel if multiple goroutines access it concurrently without proper synchronization
	// mu sync.Mutex
}

// Config holds RabbitMQ connection details.
type Config struct {
	URL string
}

// NewClient creates a new RabbitMQ client.
// It connects to RabbitMQ and sets up a channel.
func NewClient(cfg Config) (*Client, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close() // Close connection if channel creation fails
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the order queue upfront. In a real app, you might want to do this
	// more strategically or rely on exchanges and bindings.
	_, err = ch.QueueDeclare(
		"order_queue", // name
		true,          // durable (persists messages across broker restarts)
		false,         // delete when unused
		false,         // exclusive (only one connection can use it)
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare order_queue: %w", err)
	}

	log.Println("RabbitMQ client connected and order_queue declared.")

	return &Client{
		conn:    conn,
		channel: ch,
	}, nil
}

// Close closes the RabbitMQ connection and channel.
func (c *Client) Close() error {
	var errs []error
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close channel: %w", err))
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred during RabbitMQ client close: %v", errs)
	}
	return nil
}

// Publish publishes a message to rabbitmq
func (c *Client) Publish(exchange, routingKey string, body []byte) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		exchange, // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	err = ch.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	return nil
}

// ConsumeOrderEvents consumes order events
func (c *Client) ConsumeOrderEvents(messageHandler func(amqp.Delivery) error) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"orders", // name
		false,    // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			err := messageHandler(d)
			if err != nil {
				log.Printf(" [ERROR] Failed to handle message: %v", err)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}

// PublishOrderCreated publishes the order created event
func (c *Client) PublishOrderCreated(messageBody map[string]interface{}) error {
	body, err := json.Marshal(messageBody)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.Publish("order", "order.created", body)
}
