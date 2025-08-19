package events

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
	"time"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
)

type Client struct {
	client   snitchv1connect.EventServiceClient
	slogger  *slog.Logger
	session  *discordgo.Session
	handlers map[snitchv1.EventType]EventHandler
	guildID  string // The guild this bot instance operates in
}

type EventHandler func(session *discordgo.Session, event *snitchv1.Event) error

func NewClient(backendURL string, session *discordgo.Session, slogger *slog.Logger, guildID string, httpClient *http.Client) *Client {
	// Create a copy of the client for streaming with no timeout
	streamingClient := &http.Client{
		Timeout:   0,                    // No timeout for streaming connections
		Transport: httpClient.Transport, // Use same TLS config
	}

	client := snitchv1connect.NewEventServiceClient(streamingClient, backendURL)

	return &Client{
		client:   client,
		slogger:  slogger,
		session:  session,
		handlers: make(map[snitchv1.EventType]EventHandler),
		guildID:  guildID,
	}
}

func (c *Client) RegisterHandler(eventType snitchv1.EventType, handler EventHandler) {
	c.handlers[eventType] = handler
}

func (c *Client) Start(ctx context.Context) {
	c.slogger.DebugContext(ctx, "Started listening")
	go c.maintainConnection(ctx)
}

func (c *Client) maintainConnection(ctx context.Context) {
	defer c.slogger.Info("Event client connection maintenance exiting")
	retryDelay := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.connectAndListen(ctx); err != nil {
				if ctx.Err() != nil {
					return // Context cancelled, exit gracefully
				}
				c.slogger.Error(fmt.Sprintf("Event stream connection failed, retrying in %f seconds", retryDelay.Seconds()), "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
					retryDelay *= 2
					continue
				}
			}
		}
	}
}

func (c *Client) connectAndListen(ctx context.Context) error {
	// Subscribe to all event types
	req := connect.NewRequest(&snitchv1.SubscribeRequest{
		EventTypes: []snitchv1.EventType{
			snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
			snitchv1.EventType_EVENT_TYPE_REPORT_DELETED,
			snitchv1.EventType_EVENT_TYPE_USER_BANNED,
		},
	})

	// Add server ID header so backend can determine our group
	req.Header().Add("X-Server-ID", c.guildID)

	stream, err := c.client.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	c.slogger.Info("Connected to event stream")

	for stream.Receive() {
		event := stream.Msg()
		c.handleEvent(event)
	}

	if err := stream.Err(); err != nil && ctx.Err() == nil {
		c.slogger.Error("Event stream disconnected", "error", err)
		return err
	}

	return nil
}

func (c *Client) Stop() {
	// Connect streams are automatically closed when context is cancelled
	c.slogger.Info("Event client stopped")
}

func (c *Client) handleEvent(event *snitchv1.Event) {
	c.slogger.Debug("Received event", "type", event.Type, "server_id", event.ServerId)

	// Skip events from our own guild to prevent self-triggering
	if event.ServerId == c.guildID {
		c.slogger.Debug("Skipping event from own guild", "type", event.Type, "server_id", event.ServerId)
		return
	}

	handler, exists := c.handlers[event.Type]
	if !exists {
		c.slogger.Debug("No handler registered for event type", "type", event.Type)
		return
	}

	if err := handler(c.session, event); err != nil {
		c.slogger.Error("Event handler error", "type", event.Type, "error", err)
	}
}
