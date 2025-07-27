package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
)

type EventService struct {
	subscribers map[chan *snitchv1.Event]bool
	mu          sync.RWMutex
}

func NewEventService() *EventService {
	return &EventService{
		subscribers: make(map[chan *snitchv1.Event]bool),
	}
}

// Subscribe implements the streaming RPC for real-time events
func (s *EventService) Subscribe(
	ctx context.Context,
	req *connect.Request[snitchv1.SubscribeRequest],
	stream *connect.ServerStream[snitchv1.Event],
) error {
	logger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		logger = slog.Default()
	}

	logger.Info("Client subscribed to events", "event_types", req.Msg.EventTypes)

	// Create a channel for this subscriber
	eventChan := make(chan *snitchv1.Event, 256)

	// Register subscriber
	s.mu.Lock()
	s.subscribers[eventChan] = true
	s.mu.Unlock()

	// Clean up on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, eventChan)
		close(eventChan)
		s.mu.Unlock()
		logger.Info("Client unsubscribed from events", "total_subscribers", len(s.subscribers))
	}()

	// Send events to client
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				logger.Error("Failed to send event to client", "error", err)
				return err
			}
		}
	}
}

// PublishEvent broadcasts an event to all subscribers
func (s *EventService) PublishEvent(event *snitchv1.Event) error {
	return s.PublishEventWithRetry(event, 3, time.Millisecond*100)
}

// PublishEventWithRetry broadcasts an event with retry logic
func (s *EventService) PublishEventWithRetry(event *snitchv1.Event, maxRetries int, retryDelay time.Duration) error {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}

		err := s.publishEventOnce(event)
		if err == nil {
			return nil
		}

		lastErr = err
		logger := slog.Default()
		logger.Warn("Event publish attempt failed", 
			"attempt", attempt+1, 
			"max_retries", maxRetries, 
			"error", err,
			"type", event.Type)
	}

	return fmt.Errorf("event publishing failed after %d attempts: %w", maxRetries+1, lastErr)
}

// publishEventOnce attempts to publish an event once
func (s *EventService) publishEventOnce(event *snitchv1.Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger := slog.Default()
	logger.Debug("Publishing event", "type", event.Type, "server_id", event.ServerId)

	if len(s.subscribers) == 0 {
		logger.Debug("No subscribers available for event", "type", event.Type)
		return nil // Not an error - bot may not be connected
	}

	droppedCount := 0
	deliveredCount := 0

	for eventChan := range s.subscribers {
		select {
		case eventChan <- event:
			deliveredCount++
		default:
			// Channel full, count dropped events
			droppedCount++
			logger.Error("Subscriber channel full, dropping event", "type", event.Type, "server_id", event.ServerId)
		}
	}

	if droppedCount > 0 {
		logger.Error("Event delivery failed to some subscribers", 
			"type", event.Type, 
			"delivered", deliveredCount, 
			"dropped", droppedCount)
		return fmt.Errorf("event delivery failed: %d delivered, %d dropped", deliveredCount, droppedCount)
	}

	logger.Debug("Event delivered successfully", "type", event.Type, "subscribers", deliveredCount)
	return nil
}
