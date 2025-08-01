package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"snitch/internal/backend/dbclient"
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
	dbClient    *dbclient.Client
}

func NewEventService(dbClient *dbclient.Client) *EventService {
	return &EventService{
		subscribers: make(map[*subscriber]bool),
		dbClient:    dbClient,
	}
}

// Subscribe implements the streaming RPC for real-time events
func (s *EventService) Subscribe(
	ctx context.Context,
	req *connect.Request[snitchv1.SubscribeRequest],
	stream *connect.ServerStream[snitchv1.Event],
) error {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	// Get server ID from request header and look up group ID
	serverID := req.Header().Get(interceptor.ServerIDHeader)
	if serverID == "" {
		slogger.Error("Missing server ID header in subscription request")
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("server ID header is required"))
	}

	groupID, err := s.dbClient.FindGroupByServer(ctx, serverID)
	if err != nil {
		slogger.Error("Failed to find group ID for server", "server_id", serverID, "error", err)
		return connect.NewError(connect.CodeNotFound, err)
	}

	slogger.Info("Client subscribed to events", "event_types", req.Msg.EventTypes, "group_id", groupID)

	// Create a channel for this subscriber
	eventChan := make(chan *snitchv1.Event, 256)

	sub := &subscriber{
		eventChan: eventChan,
		groupID:   groupID,
	}

	// Register subscriber
	s.mu.Lock()
	s.subscribers[sub] = true
	s.mu.Unlock()

	slogger.Info("Subscribers current", "Subscribers", s.subscribers)
	// Clean up on disconnect
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, sub)
		close(eventChan)
		s.mu.Unlock()
		slogger.Info("Client unsubscribed from events", "total_subscribers", len(s.subscribers))
	}()

	// Send events to client
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				slogger.Error("Failed to send event to client", "error", err)
				return err
			}
		}
	}
}

// PublishEvent broadcasts an event to all subscribers
func (s *EventService) PublishEvent(ctx context.Context, event *snitchv1.Event) error {
	return s.PublishEventWithRetry(ctx, event, 3, time.Millisecond*100)
}

// PublishEventWithRetry broadcasts an event with retry logic
func (s *EventService) PublishEventWithRetry(ctx context.Context, event *snitchv1.Event, maxRetries int, retryDelay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}

		err := s.publishEventOnce(ctx, event)
		if err == nil {
			return nil
		}

		lastErr = err

		slogger, ok := ctxutil.Value[*slog.Logger](ctx)
		if !ok {
			slogger = slog.Default()
		}

		slogger.Warn("Event publish attempt failed",
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"error", err,
			"type", event.Type)
	}

	return fmt.Errorf("event publishing failed after %d attempts: %w", maxRetries+1, lastErr)
}

// publishEventOnce attempts to publish an event once
func (s *EventService) publishEventOnce(ctx context.Context, event *snitchv1.Event) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}
	slogger.Debug("Publishing event", "type", event.Type, "server_id", event.ServerId)

	if len(s.subscribers) == 0 {
		slogger.Debug("No subscribers available for event", "type", event.Type)
		return nil // Not an error - bot may not be connected
	}

	droppedCount := 0
	deliveredCount := 0
	filteredCount := 0

	for sub := range s.subscribers {
		slogger.Info("Delivering message to subscriber", "Group ID", sub.groupID)
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
			slogger.Error("Subscriber channel full, dropping event", "type", event.Type, "server_id", event.ServerId, "group_id", event.GroupId)
		}
	}

	if droppedCount > 0 {
		slogger.Error("Event delivery failed to some subscribers",
			"type", event.Type,
			"delivered", deliveredCount,
			"dropped", droppedCount,
			"filtered", filteredCount)
		return fmt.Errorf("event delivery failed: %d delivered, %d dropped", deliveredCount, droppedCount)
	}

	slogger.Debug("Event delivered successfully", "type", event.Type, "delivered", deliveredCount, "filtered", filteredCount)
	return nil
}
