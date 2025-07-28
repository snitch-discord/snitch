package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"snitch/internal/backend/metadata"
	"snitch/internal/backend/service/interceptor"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
)

type subscriber struct {
	eventChan chan *snitchv1.Event
	groupID   string
}

type EventService struct {
	subscribers map[*subscriber]bool
	mu          sync.RWMutex
	metadataDB  *sql.DB
}

func NewEventService(metadataDB *sql.DB) *EventService {
	return &EventService{
		subscribers: make(map[*subscriber]bool),
		metadataDB:  metadataDB,
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

	// Get server ID from request header and look up group ID
	serverID := req.Header().Get(interceptor.ServerIDHeader)
	if serverID == "" {
		logger.Error("Missing server ID header in subscription request")
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("server ID header is required"))
	}

	groupID, err := metadata.FindGroupIDByServerID(ctx, s.metadataDB, serverID)
	if err != nil {
		logger.Error("Failed to find group ID for server", "server_id", serverID, "error", err)
		return connect.NewError(connect.CodeNotFound, err)
	}

	logger.Info("Client subscribed to events", "event_types", req.Msg.EventTypes, "group_id", groupID.String())

	// Create a channel for this subscriber
	eventChan := make(chan *snitchv1.Event, 256)

	sub := &subscriber{
		eventChan: eventChan,
		groupID:   groupID.String(),
	}

	// Register subscriber
	s.mu.Lock()
	s.subscribers[sub] = true
	s.mu.Unlock()

	logger.Info("Subscribers current", "Subscribers", s.subscribers)
	// Clean up on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, sub)
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
	filteredCount := 0

	for sub := range s.subscribers {
		logger.Info("Delivering message to subscriber", "Group ID", sub.groupID)
		// Filter by group - only send events to subscribers in the same group
		if sub.groupID != event.GroupId {
			filteredCount++
			continue
		}

		select {
		case sub.eventChan <- event:
			deliveredCount++
		default:
			// Channel full, count dropped events
			droppedCount++
			logger.Error("Subscriber channel full, dropping event", "type", event.Type, "server_id", event.ServerId, "group_id", event.GroupId)
		}
	}

	if droppedCount > 0 {
		logger.Error("Event delivery failed to some subscribers",
			"type", event.Type,
			"delivered", deliveredCount,
			"dropped", droppedCount,
			"filtered", filteredCount)
		return fmt.Errorf("event delivery failed: %d delivered, %d dropped", deliveredCount, droppedCount)
	}

	logger.Debug("Event delivered successfully", "type", event.Type, "delivered", deliveredCount, "filtered", filteredCount)
	return nil
}
