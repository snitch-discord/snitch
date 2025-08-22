package events

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"snitch/internal/bot/auth"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
)

type Client struct {
	eventClient      snitchv1connect.EventServiceClient
	registerClient   snitchv1connect.RegistrarServiceClient
	slogger          *slog.Logger
	session          *discordgo.Session
	handlers         map[snitchv1.EventType]EventHandler
	tokenGenerator   *auth.TokenGenerator
	// Group-based subscriptions for efficiency
	groupSubscriptions map[string]context.CancelFunc // groupID -> cancel function
	serverToGroup      map[string]string             // serverID -> groupID
	mu                 sync.RWMutex
}

type EventHandler func(session *discordgo.Session, event *snitchv1.SubscribeResponse) error

func NewClient(backendURL string, jwtSecret string, session *discordgo.Session, slogger *slog.Logger, httpClient *http.Client) *Client {
	streamingClient := &http.Client{
		Timeout:   0,                    // No timeout for streaming connections
		Transport: httpClient.Transport, // Use same TLS config
	}

	eventClient := snitchv1connect.NewEventServiceClient(streamingClient, backendURL)
	registerClient := snitchv1connect.NewRegistrarServiceClient(streamingClient, backendURL)
	tokenGenerator := auth.NewTokenGenerator(jwtSecret)

	return &Client{
		eventClient:        eventClient,
		registerClient:     registerClient,
		slogger:            slogger,
		session:            session,
		handlers:           make(map[snitchv1.EventType]EventHandler),
		tokenGenerator:     tokenGenerator,
		groupSubscriptions: make(map[string]context.CancelFunc),
		serverToGroup:      make(map[string]string),
	}
}

func (c *Client) RegisterHandler(eventType snitchv1.EventType, handler EventHandler) {
	c.handlers[eventType] = handler
}

func (c *Client) Start(ctx context.Context) {
	c.slogger.DebugContext(ctx, "Event client started")
}

// AddServer adds a server to the event subscription (group-based)
func (c *Client) AddServer(ctx context.Context, serverID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if server is already tracked
	if groupID, exists := c.serverToGroup[serverID]; exists {
		c.slogger.Debug("Server already subscribed", "server_id", serverID, "group_id", groupID)
		return nil
	}

	hasGroup, err := c.serverHasGroup(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to check if server %s has group", serverID)
	}

	if !hasGroup {
		c.slogger.WarnContext(ctx, "server doesn't have a group, moving on", "server id", serverID)
		return nil
	}

	groupID, err := c.getGroupIDForServer(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get group ID for server %s: %w", serverID, err)
	}

	// Add server to group mapping
	c.serverToGroup[serverID] = groupID

	// Start group subscription if this is the first server in the group
	if c.countServersInGroup(groupID) == 1 {
		subCtx, cancel := context.WithCancel(ctx)
		c.groupSubscriptions[groupID] = cancel
		go c.maintainGroupConnection(subCtx, groupID, serverID)
		c.slogger.Info("Started group subscription", "group_id", groupID, "server_id", serverID)
	} else {
		c.slogger.Info("Added server to existing group subscription", "server_id", serverID, "group_id", groupID)
	}

	return nil
}

// RemoveServer removes a server from the event subscription (group-based)
func (c *Client) RemoveServer(serverID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	groupID, exists := c.serverToGroup[serverID]
	if !exists {
		c.slogger.Debug("Server not found in subscriptions", "server_id", serverID)
		return
	}

	// Remove server from group mapping
	delete(c.serverToGroup, serverID)

	// If this was the last server in the group, stop the group subscription
	if c.countServersInGroup(groupID) == 0 {
		if cancel, exists := c.groupSubscriptions[groupID]; exists {
			cancel()
			delete(c.groupSubscriptions, groupID)
			c.slogger.Info("Stopped group subscription", "group_id", groupID, "server_id", serverID)
		}
	} else {
		c.slogger.Info("Removed server from group subscription", "server_id", serverID, "group_id", groupID, "remaining_servers", c.countServersInGroup(groupID))
	}
}

// GetSubscribedServers returns the list of currently subscribed server IDs
func (c *Client) GetSubscribedServers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	servers := make([]string, 0, len(c.serverToGroup))
	for serverID := range c.serverToGroup {
		servers = append(servers, serverID)
	}
	return servers
}

// countServersInGroup returns the number of servers currently in a group
// Note: this method assumes the mutex is already held by the caller
func (c *Client) countServersInGroup(groupID string) int {
	count := 0
	for _, serverGroupID := range c.serverToGroup {
		if serverGroupID == groupID {
			count++
		}
	}
	return count
}

func (c *Client) getGroupIDForServer(ctx context.Context, serverID string) (string, error) {
	req := connect.NewRequest(&snitchv1.GetGroupForServerRequest{
		ServerId: serverID,
	})

	getGroupIdResponse, err := c.registerClient.GetGroupForServer(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get group id for server %s: %w", serverID, err)
	}

	return getGroupIdResponse.Msg.GroupId, nil
}

func (c *Client) serverHasGroup(ctx context.Context, serverID string) (bool, error) {
	req := connect.NewRequest(&snitchv1.HasGroupRequest{
		ServerId: serverID,
	})

	hasGroupResponse, err := c.registerClient.HasGroup(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to check if server %s is in a group. message: %w", serverID, err)
	}

	return hasGroupResponse.Msg.HasGroup, nil
}

func (c *Client) maintainGroupConnection(ctx context.Context, groupID, serverID string) {
	defer c.slogger.Info("Group connection maintenance exiting", "group_id", groupID)
	retryDelay := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.connectAndListenForGroup(ctx, groupID, serverID); err != nil {
				if ctx.Err() != nil {
					return // Context cancelled, exit gracefully
				}
				c.slogger.Error(fmt.Sprintf("Group %s event stream failed, retrying in %f seconds", groupID, retryDelay.Seconds()), "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
					retryDelay *= 2
					if retryDelay > time.Minute {
						retryDelay = time.Minute // Cap retry delay
					}
					continue
				}
			}
		}
	}
}

func (c *Client) connectAndListenForGroup(ctx context.Context, groupID, serverID string) error {
	// Subscribe to all event types for the specific group
	req := connect.NewRequest(&snitchv1.SubscribeRequest{
		EventTypes: []snitchv1.EventType{
			snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
			snitchv1.EventType_EVENT_TYPE_REPORT_DELETED,
			snitchv1.EventType_EVENT_TYPE_USER_BANNED,
		},
		GroupId: groupID,
	})

	token, err := c.tokenGenerator.Generate(serverID, groupID)
	if err != nil {
		return fmt.Errorf("failed to generate token for server %s: %w", serverID, err)
	}
	req.Header().Add("Authorization", "Bearer "+token)

	stream, err := c.eventClient.Subscribe(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events for server %s: %w", serverID, err)
	}

	c.slogger.Info("Connected to event stream", "group_id", groupID, "server_id", serverID)

	for stream.Receive() {
		event := stream.Msg()
		c.handleEvent(event)
	}

	if err := stream.Err(); err != nil && ctx.Err() == nil {
		c.slogger.Error("Event stream disconnected", "group_id", groupID, "server_id", serverID, "error", err)
		return err
	}

	return nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel all group subscriptions
	for groupID, cancel := range c.groupSubscriptions {
		cancel()
		c.slogger.Info("Stopped group subscription", "group_id", groupID)
	}

	// Clear all maps
	c.groupSubscriptions = make(map[string]context.CancelFunc)
	c.serverToGroup = make(map[string]string)
	c.slogger.Info("Event client stopped")
}

func (c *Client) handleEvent(event *snitchv1.SubscribeResponse) {
	c.slogger.Debug("Received event", "type", event.Type, "server_id", event.ServerId)

	handler, exists := c.handlers[event.Type]
	if !exists {
		c.slogger.Debug("No handler registered for event type", "type", event.Type)
		return
	}

	if err := handler(c.session, event); err != nil {
		c.slogger.Error("Event handler error", "type", event.Type, "error", err)
	}
}
