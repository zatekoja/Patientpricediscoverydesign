package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	redisclient "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
)

// RedisEventBus implements the EventBus interface using Redis Pub/Sub
type RedisEventBus struct {
	client        *redisclient.Client
	subscriptions map[string]*redis.PubSub
	subscribers   map[string]map[chan *entities.FacilityEvent]struct{}
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewRedisEventBus creates a new Redis-based event bus
func NewRedisEventBus(client *redisclient.Client) providers.EventBus {
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisEventBus{
		client:        client,
		subscriptions: make(map[string]*redis.PubSub),
		subscribers:   make(map[string]map[chan *entities.FacilityEvent]struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Publish publishes an event to all subscribers
func (b *RedisEventBus) Publish(ctx context.Context, channel string, event *entities.FacilityEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := b.client.Client().Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("Published event to channel %s: %s", channel, event.ID)
	return nil
}

// Subscribe subscribes to events on a channel
func (b *RedisEventBus) Subscribe(ctx context.Context, channel string) (<-chan *entities.FacilityEvent, error) {
	b.mu.Lock()

	if _, exists := b.subscriptions[channel]; !exists {
		pubsub := b.client.Client().Subscribe(b.ctx, channel)
		b.subscriptions[channel] = pubsub
		go b.receiveMessages(channel, pubsub)
	}

	if b.subscribers[channel] == nil {
		b.subscribers[channel] = make(map[chan *entities.FacilityEvent]struct{})
	}

	eventChan := make(chan *entities.FacilityEvent, 100)
	b.subscribers[channel][eventChan] = struct{}{}
	subscriberCount := len(b.subscribers[channel])
	b.mu.Unlock()

	log.Printf("Subscribed to channel: %s (subscribers: %d)", channel, subscriberCount)

	go func() {
		<-ctx.Done()
		b.removeSubscriber(channel, eventChan)
	}()

	return eventChan, nil
}

// receiveMessages receives messages from Redis and broadcasts them to subscribers
func (b *RedisEventBus) receiveMessages(channel string, pubsub *redis.PubSub) {
	defer func() {
		if err := b.cleanupChannel(channel); err != nil {
			log.Printf("Failed to cleanup channel %s: %v", channel, err)
		}
	}()

	ch := pubsub.Channel()
	for {
		select {
		case <-b.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			var event entities.FacilityEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("Failed to unmarshal event from channel %s: %v", channel, err)
				continue
			}

			b.mu.RLock()
			subscribers := b.subscribers[channel]
			for subscriber := range subscribers {
				select {
				case subscriber <- &event:
				default:
					// Subscriber channel full, skip event
					log.Printf("Subscriber channel full for %s, skipping event %s", channel, event.ID)
				}
			}
			b.mu.RUnlock()
		}
	}
}

func (b *RedisEventBus) removeSubscriber(channel string, eventChan chan *entities.FacilityEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, exists := b.subscribers[channel]
	if !exists {
		return
	}

	if _, ok := subscribers[eventChan]; !ok {
		return
	}

	delete(subscribers, eventChan)
	close(eventChan)

	if len(subscribers) == 0 {
		delete(b.subscribers, channel)
		if pubsub, ok := b.subscriptions[channel]; ok {
			_ = pubsub.Close()
			delete(b.subscriptions, channel)
			log.Printf("Closed subscription to channel: %s", channel)
		}
	}
}

func (b *RedisEventBus) cleanupChannel(channel string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, exists := b.subscribers[channel]
	if exists {
		for subscriber := range subscribers {
			close(subscriber)
		}
		delete(b.subscribers, channel)
	}

	if pubsub, ok := b.subscriptions[channel]; ok {
		if err := pubsub.Close(); err != nil {
			return fmt.Errorf("failed to close subscription %s: %w", channel, err)
		}
		delete(b.subscriptions, channel)
		log.Printf("Closed subscription to channel: %s", channel)
	}

	return nil
}

// Unsubscribe unsubscribes from a channel
func (b *RedisEventBus) Unsubscribe(ctx context.Context, channel string) error {
	if err := b.cleanupChannel(channel); err != nil {
		return err
	}
	log.Printf("Unsubscribed from channel: %s", channel)
	return nil
}

// Close closes the event bus and all subscriptions
func (b *RedisEventBus) Close() error {
	b.cancel()

	b.mu.RLock()
	channels := make([]string, 0, len(b.subscriptions))
	for channel := range b.subscriptions {
		channels = append(channels, channel)
	}
	b.mu.RUnlock()

	var errs []error
	for _, channel := range channels {
		if err := b.cleanupChannel(channel); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing event bus: %v", errs)
	}

	log.Println("Event bus closed")
	return nil
}
