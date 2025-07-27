package service

import (
	"context"
	"log/slog"
	"sync"

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
func (s *EventService) PublishEvent(event *snitchv1.Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logger := slog.Default()
	logger.Debug("Publishing event", "type", event.Type, "server_id", event.ServerId)

	for eventChan := range s.subscribers {
		select {
		case eventChan <- event:
		default:
			// Channel full, skip this subscriber
			logger.Warn("Subscriber channel full, dropping event", "type", event.Type)
		}
	}
}
