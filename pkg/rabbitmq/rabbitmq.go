package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/streadway/amqp"
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

// PublishOrderCreated publishes an order creation event to the 'order_queue'.
// The message is marshaled to JSON.
func (c *Client) PublishOrderCreated(orderData map[string]interface{}) error {
	if c.channel == nil {
		return fmt.Errorf("RabbitMQ channel is not available")
	}

	// Marshal the order data to JSON
	body, err := json.Marshal(orderData)
	if err != nil {
		return fmt.Errorf("failed to marshal order data to JSON: %w", err)
	}

	// Publish the message to the 'order_queue'
	err = c.channel.Publish(
		"",          // exchange: default exchange
		"order_queue", // routing key: the queue name
		false,       // mandatory: if true, returns message if it cannot be routed
		false,       // immediate: if true, returns message if it cannot be delivered to any consumer
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			// You can add delivery mode, headers, expiration, etc. here
			DeliveryMode: amqp.Persistent, // Make message persistent
			Timestamp:    time.Now(),
		})

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf(" [x] Sent order event: %s", body)
	return nil
}

// --- Consumer logic would go here ---
// Example: ConsumeOrderEvents starts a goroutine to listen for messages on 'order_queue'.
// In a real application, this would typically be run as a separate process or a dedicated consumer goroutine.
func (c *Client) ConsumeOrderEvents(messageHandler func(msg amqp.Delivery) error) error {
	if c.channel == nil {
		return fmt.Errorf("RabbitMQ channel is not available for consumption")
	}

	// Ensure the queue exists (it should have been declared by NewClient, but good practice to re-declare)
	queue, err := c.channel.QueueDeclare(
		"order_queue", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue for consuming: %w", err)
	}

	// Start consuming messages
	msgs, err := c.channel.Consume(
		queue.Name, // queue
		"",         // consumer tag: unique identifier for the consumer
		false,      // auto-ack: set to false to manually acknowledge messages
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf(" [*] Waiting for order events. To exit press CTRL+C")

	// Start a goroutine to process messages
	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %v", msg.DeliveryTag)
			// Process the message using the provided handler
			if err := messageHandler(msg); err != nil {
				log.Printf("Error processing message %d: %v", msg.DeliveryTag, err)
				// Negative acknowledge the message to requeue it (or send to dead-letter queue)
				// Be careful with requeueing to avoid infinite loops for unprocessable messages.
				if requeueErr := msg.Nack(false, true); requeueErr != nil {
					log.Printf("Error nacking message %d: %v", msg.DeliveryTag, requeueErr)
				}
			} else {
				// Manually acknowledge the message upon successful processing
				if ackErr := msg.Ack(false); ackErr != nil {
					log.Printf("Error acking message %d: %v", msg.DeliveryTag, ackErr)
				}
			}
		}
	}()

	return nil
}

// Example of a handler function that might be passed to ConsumeOrderEvents
// This is just a placeholder and would be implemented by the application logic.
func HandleOrderMessage(msg amqp.Delivery) error {
	log.Printf("Processing order message: %s", string(msg.Body))
	// Here you would parse the message body (e.g., order details)
	// and perform actions like:
	// - Update inventory
	// - Send confirmation email/SMS
	// - Trigger other workflows

	// Simulate processing time
	time.Sleep(2 * time.Second)

	// If processing is successful, the message will be acknowledged.
	// If an error occurs, the message will be NACKed and potentially requeued.
	return nil // Returning nil indicates successful processing
}

