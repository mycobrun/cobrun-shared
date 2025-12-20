// Package messaging provides messaging client utilities.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// ServiceBusConfig holds Service Bus configuration.
type ServiceBusConfig struct {
	Namespace        string
	ConnectionString string // Optional - if empty, uses managed identity
}

// ServiceBusClient wraps the Azure Service Bus client.
type ServiceBusClient struct {
	client *azservicebus.Client
	config ServiceBusConfig
}

// NewServiceBusClient creates a new Service Bus client.
func NewServiceBusClient(ctx context.Context, config ServiceBusConfig) (*ServiceBusClient, error) {
	var client *azservicebus.Client
	var err error

	if config.ConnectionString != "" {
		// Use connection string
		client, err = azservicebus.NewClientFromConnectionString(config.ConnectionString, nil)
	} else {
		// Use managed identity
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create credential: %w", credErr)
		}
		fullyQualifiedNamespace := fmt.Sprintf("%s.servicebus.windows.net", config.Namespace)
		client, err = azservicebus.NewClient(fullyQualifiedNamespace, cred, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create service bus client: %w", err)
	}

	return &ServiceBusClient{
		client: client,
		config: config,
	}, nil
}

// Close closes the client.
func (c *ServiceBusClient) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}

// Publisher publishes messages to a queue or topic.
type Publisher struct {
	sender *azservicebus.Sender
}

// NewQueuePublisher creates a publisher for a queue.
func (c *ServiceBusClient) NewQueuePublisher(queueName string) (*Publisher, error) {
	sender, err := c.client.NewSender(queueName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender for queue %s: %w", queueName, err)
	}
	return &Publisher{sender: sender}, nil
}

// NewTopicPublisher creates a publisher for a topic.
func (c *ServiceBusClient) NewTopicPublisher(topicName string) (*Publisher, error) {
	sender, err := c.client.NewSender(topicName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender for topic %s: %w", topicName, err)
	}
	return &Publisher{sender: sender}, nil
}

// Close closes the publisher.
func (p *Publisher) Close(ctx context.Context) error {
	return p.sender.Close(ctx)
}

// Send sends a message.
func (p *Publisher) Send(ctx context.Context, msg *Message) error {
	sbMsg := &azservicebus.Message{
		Body:          msg.Body,
		ContentType:   &msg.ContentType,
		CorrelationID: &msg.CorrelationID,
		MessageID:     &msg.ID,
	}

	if msg.Subject != "" {
		sbMsg.Subject = &msg.Subject
	}

	if msg.SessionID != "" {
		sbMsg.SessionID = &msg.SessionID
	}

	if msg.ScheduledEnqueueTime != nil {
		sbMsg.ScheduledEnqueueTime = msg.ScheduledEnqueueTime
	}

	if len(msg.Properties) > 0 {
		sbMsg.ApplicationProperties = make(map[string]any)
		for k, v := range msg.Properties {
			sbMsg.ApplicationProperties[k] = v
		}
	}

	return p.sender.SendMessage(ctx, sbMsg, nil)
}

// SendJSON sends a JSON-encoded message.
func (p *Publisher) SendJSON(ctx context.Context, id string, data interface{}, opts ...MessageOption) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := &Message{
		ID:          id,
		Body:        body,
		ContentType: "application/json",
		Properties:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(msg)
	}

	return p.Send(ctx, msg)
}

// SendBatch sends a batch of messages.
func (p *Publisher) SendBatch(ctx context.Context, messages []*Message) error {
	batch, err := p.sender.NewMessageBatch(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create message batch: %w", err)
	}

	for _, msg := range messages {
		sbMsg := &azservicebus.Message{
			Body:          msg.Body,
			ContentType:   &msg.ContentType,
			CorrelationID: &msg.CorrelationID,
			MessageID:     &msg.ID,
		}

		if msg.Subject != "" {
			sbMsg.Subject = &msg.Subject
		}

		if err := batch.AddMessage(sbMsg, nil); err != nil {
			// Batch is full, send what we have and create a new batch
			if err := p.sender.SendMessageBatch(ctx, batch, nil); err != nil {
				return fmt.Errorf("failed to send batch: %w", err)
			}

			batch, err = p.sender.NewMessageBatch(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to create new batch: %w", err)
			}

			if err := batch.AddMessage(sbMsg, nil); err != nil {
				return fmt.Errorf("message too large for batch: %w", err)
			}
		}
	}

	// Send remaining messages
	if batch.NumMessages() > 0 {
		if err := p.sender.SendMessageBatch(ctx, batch, nil); err != nil {
			return fmt.Errorf("failed to send final batch: %w", err)
		}
	}

	return nil
}

// ScheduleMessage schedules a message for later delivery.
func (p *Publisher) ScheduleMessage(ctx context.Context, msg *Message, scheduledTime time.Time) (int64, error) {
	sbMsg := &azservicebus.Message{
		Body:          msg.Body,
		ContentType:   &msg.ContentType,
		CorrelationID: &msg.CorrelationID,
		MessageID:     &msg.ID,
	}

	seqNumbers, err := p.sender.ScheduleMessages(ctx, []*azservicebus.Message{sbMsg}, scheduledTime, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to schedule message: %w", err)
	}

	return seqNumbers[0], nil
}

// CancelScheduledMessage cancels a scheduled message.
func (p *Publisher) CancelScheduledMessage(ctx context.Context, sequenceNumber int64) error {
	return p.sender.CancelScheduledMessages(ctx, []int64{sequenceNumber}, nil)
}

// Consumer consumes messages from a queue or subscription.
type Consumer struct {
	receiver *azservicebus.Receiver
}

// NewQueueConsumer creates a consumer for a queue.
func (c *ServiceBusClient) NewQueueConsumer(queueName string, opts *azservicebus.ReceiverOptions) (*Consumer, error) {
	receiver, err := c.client.NewReceiverForQueue(queueName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create receiver for queue %s: %w", queueName, err)
	}
	return &Consumer{receiver: receiver}, nil
}

// NewSubscriptionConsumer creates a consumer for a topic subscription.
func (c *ServiceBusClient) NewSubscriptionConsumer(topicName, subscriptionName string, opts *azservicebus.ReceiverOptions) (*Consumer, error) {
	receiver, err := c.client.NewReceiverForSubscription(topicName, subscriptionName, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create receiver for subscription %s/%s: %w", topicName, subscriptionName, err)
	}
	return &Consumer{receiver: receiver}, nil
}

// Close closes the consumer.
func (c *Consumer) Close(ctx context.Context) error {
	return c.receiver.Close(ctx)
}

// Receive receives messages.
func (c *Consumer) Receive(ctx context.Context, maxMessages int) ([]*ReceivedMessage, error) {
	messages, err := c.receiver.ReceiveMessages(ctx, maxMessages, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to receive messages: %w", err)
	}

	result := make([]*ReceivedMessage, len(messages))
	for i, msg := range messages {
		result[i] = &ReceivedMessage{
			raw:     msg,
			Message: fromServiceBusMessage(msg),
		}
	}

	return result, nil
}

// Complete completes a message.
func (c *Consumer) Complete(ctx context.Context, msg *ReceivedMessage) error {
	return c.receiver.CompleteMessage(ctx, msg.raw, nil)
}

// Abandon abandons a message (returns it to the queue).
func (c *Consumer) Abandon(ctx context.Context, msg *ReceivedMessage) error {
	return c.receiver.AbandonMessage(ctx, msg.raw, nil)
}

// DeadLetter moves a message to the dead letter queue.
func (c *Consumer) DeadLetter(ctx context.Context, msg *ReceivedMessage, reason, description string) error {
	opts := &azservicebus.DeadLetterOptions{
		ErrorDescription: &description,
		Reason:           &reason,
	}
	return c.receiver.DeadLetterMessage(ctx, msg.raw, opts)
}

// Defer defers a message.
func (c *Consumer) Defer(ctx context.Context, msg *ReceivedMessage) error {
	return c.receiver.DeferMessage(ctx, msg.raw, nil)
}

// Message represents a message to be sent.
type Message struct {
	ID                   string
	Body                 []byte
	ContentType          string
	CorrelationID        string
	Subject              string
	SessionID            string
	ScheduledEnqueueTime *time.Time
	Properties           map[string]string
}

// MessageOption configures a message.
type MessageOption func(*Message)

// WithCorrelationID sets the correlation ID.
func WithCorrelationID(id string) MessageOption {
	return func(m *Message) {
		m.CorrelationID = id
	}
}

// WithSubject sets the subject.
func WithSubject(subject string) MessageOption {
	return func(m *Message) {
		m.Subject = subject
	}
}

// WithSessionID sets the session ID.
func WithSessionID(sessionID string) MessageOption {
	return func(m *Message) {
		m.SessionID = sessionID
	}
}

// WithProperty sets a custom property.
func WithProperty(key, value string) MessageOption {
	return func(m *Message) {
		if m.Properties == nil {
			m.Properties = make(map[string]string)
		}
		m.Properties[key] = value
	}
}

// WithScheduledTime sets the scheduled delivery time.
func WithScheduledTime(t time.Time) MessageOption {
	return func(m *Message) {
		m.ScheduledEnqueueTime = &t
	}
}

// ReceivedMessage represents a received message.
type ReceivedMessage struct {
	raw *azservicebus.ReceivedMessage
	Message
	EnqueuedTime   time.Time
	DeliveryCount  uint32
	SequenceNumber int64
}

// UnmarshalJSON unmarshals the message body as JSON.
func (m *ReceivedMessage) UnmarshalJSON(v interface{}) error {
	return json.Unmarshal(m.Body, v)
}

func fromServiceBusMessage(msg *azservicebus.ReceivedMessage) Message {
	m := Message{
		Body: msg.Body,
	}

	if msg.MessageID != nil {
		m.ID = *msg.MessageID
	}
	if msg.ContentType != nil {
		m.ContentType = *msg.ContentType
	}
	if msg.CorrelationID != nil {
		m.CorrelationID = *msg.CorrelationID
	}
	if msg.Subject != nil {
		m.Subject = *msg.Subject
	}
	if msg.SessionID != nil {
		m.SessionID = *msg.SessionID
	}

	if msg.ApplicationProperties != nil {
		m.Properties = make(map[string]string)
		for k, v := range msg.ApplicationProperties {
			if s, ok := v.(string); ok {
				m.Properties[k] = s
			}
		}
	}

	return m
}

// MessageHandler is a function that handles received messages.
type MessageHandler func(ctx context.Context, msg *ReceivedMessage) error

// StartConsumer starts a consumer loop that processes messages.
func (c *Consumer) StartConsumer(ctx context.Context, handler MessageHandler, maxConcurrent int) error {
	sem := make(chan struct{}, maxConcurrent)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		messages, err := c.Receive(ctx, maxConcurrent)
		if err != nil {
			// Log error and continue
			continue
		}

		for _, msg := range messages {
			sem <- struct{}{} // Acquire semaphore

			go func(m *ReceivedMessage) {
				defer func() { <-sem }() // Release semaphore

				if err := handler(ctx, m); err != nil {
					// Handler failed, abandon the message
					c.Abandon(ctx, m)
				} else {
					// Handler succeeded, complete the message
					c.Complete(ctx, m)
				}
			}(msg)
		}
	}
}
