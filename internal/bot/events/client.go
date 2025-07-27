package events

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
)

type Client struct {
	client   snitchv1connect.EventServiceClient
	logger   *slog.Logger
	session  *discordgo.Session
	handlers map[snitchv1.EventType]EventHandler
}

type EventHandler func(session *discordgo.Session, event *snitchv1.Event) error

func NewClient(backendURL string, session *discordgo.Session, logger *slog.Logger) *Client {
	httpClient := &http.Client{
		Timeout: 0, // No timeout for streaming connections
	}

	client := snitchv1connect.NewEventServiceClient(httpClient, backendURL)

	return &Client{
		client:   client,
		logger:   logger,
		session:  session,
		handlers: make(map[snitchv1.EventType]EventHandler),
	}
}

func (c *Client) RegisterHandler(eventType snitchv1.EventType, handler EventHandler) {
	c.handlers[eventType] = handler
}

func (c *Client) Start(ctx context.Context) error {
	// Subscribe to all event types
	req := connect.NewRequest(&snitchv1.SubscribeRequest{
		EventTypes: []snitchv1.EventType{
			snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
			snitchv1.EventType_EVENT_TYPE_REPORT_DELETED,
			snitchv1.EventType_EVENT_TYPE_USER_BANNED,
		},
	})

	stream, err := c.client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	c.logger.Info("Connected to event stream")

	go func() {
		defer c.logger.Info("Event stream goroutine exiting")
		
		for stream.Receive() {
			event := stream.Msg()
			c.handleEvent(event)
		}

		if err := stream.Err(); err != nil && ctx.Err() == nil {
			c.logger.Error("Event stream error", "error", err)
		}
	}()

	return nil
}

func (c *Client) Stop() {
	// Connect streams are automatically closed when context is cancelled
	c.logger.Info("Event client stopped")
}

func (c *Client) handleEvent(event *snitchv1.Event) {
	handler, exists := c.handlers[event.Type]
	if !exists {
		c.logger.Debug("No handler registered for event type", "type", event.Type)
		return
	}

	if err := handler(c.session, event); err != nil {
		c.logger.Error("Event handler error", "type", event.Type, "error", err)
	}
}
