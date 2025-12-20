// Package messaging provides messaging client utilities.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

// EventHubsConfig holds Event Hubs configuration.
type EventHubsConfig struct {
	Namespace        string
	EventHubName     string
	ConnectionString string // Optional - if empty, uses managed identity
	ConsumerGroup    string // Default is "$Default"
}

// EventHubsProducer produces events to Event Hubs.
type EventHubsProducer struct {
	client *azeventhubs.ProducerClient
	config EventHubsConfig
}

// NewEventHubsProducer creates a new Event Hubs producer.
func NewEventHubsProducer(ctx context.Context, config EventHubsConfig) (*EventHubsProducer, error) {
	var client *azeventhubs.ProducerClient
	var err error

	if config.ConnectionString != "" {
		client, err = azeventhubs.NewProducerClientFromConnectionString(
			config.ConnectionString,
			config.EventHubName,
			nil,
		)
	} else {
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create credential: %w", credErr)
		}
		fullyQualifiedNamespace := fmt.Sprintf("%s.servicebus.windows.net", config.Namespace)
		client, err = azeventhubs.NewProducerClient(
			fullyQualifiedNamespace,
			config.EventHubName,
			cred,
			nil,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create event hubs producer: %w", err)
	}

	return &EventHubsProducer{
		client: client,
		config: config,
	}, nil
}

// Close closes the producer.
func (p *EventHubsProducer) Close(ctx context.Context) error {
	return p.client.Close(ctx)
}

// Send sends a single event.
func (p *EventHubsProducer) Send(ctx context.Context, event *Event) error {
	batch, err := p.client.NewEventDataBatch(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create event batch: %w", err)
	}

	eventData := &azeventhubs.EventData{
		Body: event.Body,
	}

	if len(event.Properties) > 0 {
		eventData.Properties = make(map[string]any)
		for k, v := range event.Properties {
			eventData.Properties[k] = v
		}
	}

	if event.ContentType != "" {
		eventData.ContentType = &event.ContentType
	}

	if err := batch.AddEventData(eventData, nil); err != nil {
		return fmt.Errorf("failed to add event to batch: %w", err)
	}

	return p.client.SendEventDataBatch(ctx, batch, nil)
}

// SendJSON sends a JSON-encoded event.
func (p *EventHubsProducer) SendJSON(ctx context.Context, data interface{}, opts ...EventOption) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	event := &Event{
		Body:        body,
		ContentType: "application/json",
		Properties:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(event)
	}

	return p.Send(ctx, event)
}

// SendBatch sends a batch of events.
func (p *EventHubsProducer) SendBatch(ctx context.Context, events []*Event) error {
	batch, err := p.client.NewEventDataBatch(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create event batch: %w", err)
	}

	for _, event := range events {
		eventData := &azeventhubs.EventData{
			Body: event.Body,
		}

		if len(event.Properties) > 0 {
			eventData.Properties = make(map[string]any)
			for k, v := range event.Properties {
				eventData.Properties[k] = v
			}
		}

		if err := batch.AddEventData(eventData, nil); err != nil {
			// Batch is full, send and create new
			if err := p.client.SendEventDataBatch(ctx, batch, nil); err != nil {
				return fmt.Errorf("failed to send batch: %w", err)
			}

			batch, err = p.client.NewEventDataBatch(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to create new batch: %w", err)
			}

			if err := batch.AddEventData(eventData, nil); err != nil {
				return fmt.Errorf("event too large for batch: %w", err)
			}
		}
	}

	if batch.NumEvents() > 0 {
		if err := p.client.SendEventDataBatch(ctx, batch, nil); err != nil {
			return fmt.Errorf("failed to send final batch: %w", err)
		}
	}

	return nil
}

// SendToPartition sends events to a specific partition.
func (p *EventHubsProducer) SendToPartition(ctx context.Context, partitionID string, events []*Event) error {
	opts := &azeventhubs.EventDataBatchOptions{
		PartitionID: &partitionID,
	}

	batch, err := p.client.NewEventDataBatch(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create event batch: %w", err)
	}

	for _, event := range events {
		eventData := &azeventhubs.EventData{
			Body: event.Body,
		}

		if err := batch.AddEventData(eventData, nil); err != nil {
			return fmt.Errorf("failed to add event to batch: %w", err)
		}
	}

	return p.client.SendEventDataBatch(ctx, batch, nil)
}

// GetPartitionIDs returns the partition IDs.
func (p *EventHubsProducer) GetPartitionIDs(ctx context.Context) ([]string, error) {
	props, err := p.client.GetEventHubProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get event hub properties: %w", err)
	}
	return props.PartitionIDs, nil
}

// Event represents an event to be sent.
type Event struct {
	Body         []byte
	ContentType  string
	PartitionKey string
	Properties   map[string]string
}

// EventOption configures an event.
type EventOption func(*Event)

// WithEventProperty sets a custom property.
func WithEventProperty(key, value string) EventOption {
	return func(e *Event) {
		if e.Properties == nil {
			e.Properties = make(map[string]string)
		}
		e.Properties[key] = value
	}
}

// WithPartitionKey sets the partition key.
func WithPartitionKey(key string) EventOption {
	return func(e *Event) {
		e.PartitionKey = key
	}
}

// ReceivedEvent represents a received event.
type ReceivedEvent struct {
	Body             []byte
	ContentType      string
	Properties       map[string]string
	EnqueuedTime     time.Time
	SequenceNumber   int64
	Offset           int64
	PartitionKey     string
	PartitionID      string
}

// UnmarshalJSON unmarshals the event body as JSON.
func (e *ReceivedEvent) UnmarshalJSON(v interface{}) error {
	return json.Unmarshal(e.Body, v)
}

// EventHandler is a function that handles received events.
type EventHandler func(ctx context.Context, event *ReceivedEvent) error

// EventHubsConsumerConfig holds consumer configuration.
type EventHubsConsumerConfig struct {
	EventHubsConfig
	StorageConnectionString string // For checkpoint store
	StorageContainerName    string
}

// EventHubsConsumer consumes events using the processor pattern.
type EventHubsConsumer struct {
	processor *azeventhubs.Processor
	config    EventHubsConsumerConfig
	handlers  map[string]EventHandler
	mu        sync.RWMutex
}

// NewEventHubsConsumer creates a new Event Hubs consumer with checkpoint store.
func NewEventHubsConsumer(ctx context.Context, config EventHubsConsumerConfig) (*EventHubsConsumer, error) {
	// Create checkpoint store
	containerClient, err := container.NewClientFromConnectionString(
		config.StorageConnectionString,
		config.StorageContainerName,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container client: %w", err)
	}

	checkpointStore, err := checkpoints.NewBlobStore(containerClient, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpoint store: %w", err)
	}

	// Create consumer client
	var consumerClient *azeventhubs.ConsumerClient

	consumerGroup := config.ConsumerGroup
	if consumerGroup == "" {
		consumerGroup = azeventhubs.DefaultConsumerGroup
	}

	if config.ConnectionString != "" {
		consumerClient, err = azeventhubs.NewConsumerClientFromConnectionString(
			config.ConnectionString,
			config.EventHubName,
			consumerGroup,
			nil,
		)
	} else {
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create credential: %w", credErr)
		}
		fullyQualifiedNamespace := fmt.Sprintf("%s.servicebus.windows.net", config.Namespace)
		consumerClient, err = azeventhubs.NewConsumerClient(
			fullyQualifiedNamespace,
			config.EventHubName,
			consumerGroup,
			cred,
			nil,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create consumer client: %w", err)
	}

	// Create processor
	processor, err := azeventhubs.NewProcessor(consumerClient, checkpointStore, nil)
	if err != nil {
		consumerClient.Close(ctx)
		return nil, fmt.Errorf("failed to create processor: %w", err)
	}

	return &EventHubsConsumer{
		processor: processor,
		config:    config,
		handlers:  make(map[string]EventHandler),
	}, nil
}

// Start starts the consumer with the given handler.
func (c *EventHubsConsumer) Start(ctx context.Context, handler EventHandler) error {
	go func() {
		for {
			partitionClient := c.processor.NextPartitionClient(ctx)
			if partitionClient == nil {
				break
			}

			go c.processPartition(ctx, partitionClient, handler)
		}
	}()

	return c.processor.Run(ctx)
}

func (c *EventHubsConsumer) processPartition(ctx context.Context, partitionClient *azeventhubs.ProcessorPartitionClient, handler EventHandler) {
	defer partitionClient.Close(ctx)

	for {
		events, err := partitionClient.ReceiveEvents(ctx, 100, nil)
		if err != nil {
			return
		}

		for _, event := range events {
			receivedEvent := &ReceivedEvent{
				Body:           event.Body,
				EnqueuedTime:   *event.EnqueuedTime,
				SequenceNumber: event.SequenceNumber,
				Offset:         event.Offset,
				PartitionID:    partitionClient.PartitionID(),
			}

			if event.ContentType != nil {
				receivedEvent.ContentType = *event.ContentType
			}

			if event.PartitionKey != nil {
				receivedEvent.PartitionKey = *event.PartitionKey
			}

			if event.Properties != nil {
				receivedEvent.Properties = make(map[string]string)
				for k, v := range event.Properties {
					if s, ok := v.(string); ok {
						receivedEvent.Properties[k] = s
					}
				}
			}

			if err := handler(ctx, receivedEvent); err != nil {
				// Log error but continue processing
				continue
			}
		}

		// Update checkpoint after processing batch
		if len(events) > 0 {
			if err := partitionClient.UpdateCheckpoint(ctx, events[len(events)-1], nil); err != nil {
				// Log checkpoint error
				continue
			}
		}
	}
}

// Close closes the consumer.
func (c *EventHubsConsumer) Close(ctx context.Context) error {
	// Processor doesn't have a close method - it stops when context is cancelled
	return nil
}

// SimpleEventHubsConsumer is a simpler consumer without checkpointing.
type SimpleEventHubsConsumer struct {
	client        *azeventhubs.ConsumerClient
	config        EventHubsConfig
	consumerGroup string
}

// NewSimpleEventHubsConsumer creates a simple consumer.
func NewSimpleEventHubsConsumer(ctx context.Context, config EventHubsConfig) (*SimpleEventHubsConsumer, error) {
	var client *azeventhubs.ConsumerClient
	var err error

	consumerGroup := config.ConsumerGroup
	if consumerGroup == "" {
		consumerGroup = azeventhubs.DefaultConsumerGroup
	}

	if config.ConnectionString != "" {
		client, err = azeventhubs.NewConsumerClientFromConnectionString(
			config.ConnectionString,
			config.EventHubName,
			consumerGroup,
			nil,
		)
	} else {
		cred, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			return nil, fmt.Errorf("failed to create credential: %w", credErr)
		}
		fullyQualifiedNamespace := fmt.Sprintf("%s.servicebus.windows.net", config.Namespace)
		client, err = azeventhubs.NewConsumerClient(
			fullyQualifiedNamespace,
			config.EventHubName,
			consumerGroup,
			cred,
			nil,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create consumer client: %w", err)
	}

	return &SimpleEventHubsConsumer{
		client:        client,
		config:        config,
		consumerGroup: consumerGroup,
	}, nil
}

// ReceiveFromPartition receives events from a specific partition.
func (c *SimpleEventHubsConsumer) ReceiveFromPartition(ctx context.Context, partitionID string, startPosition azeventhubs.StartPosition, count int) ([]*ReceivedEvent, error) {
	partitionClient, err := c.client.NewPartitionClient(partitionID, &azeventhubs.PartitionClientOptions{
		StartPosition: startPosition,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create partition client: %w", err)
	}
	defer partitionClient.Close(ctx)

	events, err := partitionClient.ReceiveEvents(ctx, count, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to receive events: %w", err)
	}

	result := make([]*ReceivedEvent, len(events))
	for i, event := range events {
		result[i] = &ReceivedEvent{
			Body:           event.Body,
			PartitionID:    partitionID,
		}

		if event.EnqueuedTime != nil {
			result[i].EnqueuedTime = *event.EnqueuedTime
		}
		result[i].SequenceNumber = event.SequenceNumber
		result[i].Offset = event.Offset
		if event.ContentType != nil {
			result[i].ContentType = *event.ContentType
		}
	}

	return result, nil
}

// Close closes the consumer.
func (c *SimpleEventHubsConsumer) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}

// GetPartitionIDs returns partition IDs.
func (c *SimpleEventHubsConsumer) GetPartitionIDs(ctx context.Context) ([]string, error) {
	props, err := c.client.GetEventHubProperties(ctx, nil)
	if err != nil {
		return nil, err
	}
	return props.PartitionIDs, nil
}
