package events

import (
	"log/slog"
	"testing"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"github.com/bwmarrin/discordgo"
)

func TestClient_Creation(t *testing.T) {
	// Test that client can be created without errors
	session := &discordgo.Session{}
	logger := slog.Default()
	
	client := NewClient("http://localhost:4200", session, logger)
	
	if client.client == nil {
		t.Error("Connect client should not be nil")
	}
	
	if client.logger != logger {
		t.Error("Logger should be set correctly")
	}
	
	if client.session != session {
		t.Error("Discord session should be set correctly")
	}
	
	if client.handlers == nil {
		t.Error("Handlers map should be initialized")
	}
}

func TestClient_RegisterHandler(t *testing.T) {
	session := &discordgo.Session{}
	logger := slog.Default()
	client := NewClient("http://localhost:4200", session, logger)
	
	// Test handler registration
	handlerCalled := false
	testHandler := func(session *discordgo.Session, event *snitchv1.Event) error {
		handlerCalled = true
		return nil
	}
	
	client.RegisterHandler(snitchv1.EventType_EVENT_TYPE_REPORT_CREATED, testHandler)
	
	// Verify handler was registered
	if len(client.handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(client.handlers))
	}
	
	// Test that the handler can be called
	testEvent := &snitchv1.Event{
		Type: snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
	}
	
	client.handleEvent(testEvent)
	
	if !handlerCalled {
		t.Error("Handler should have been called")
	}
}


func TestClient_HandlerNotFound(t *testing.T) {
	session := &discordgo.Session{}
	logger := slog.Default()
	client := NewClient("http://localhost:4200", session, logger)
	
	// Test handling an event type with no registered handler
	testEvent := &snitchv1.Event{
		Type: snitchv1.EventType_EVENT_TYPE_USER_BANNED,
	}
	
	// This should not panic
	client.handleEvent(testEvent)
	
	// No assertion needed - just testing that it doesn't panic
}

func TestClient_MultipleHandlers(t *testing.T) {
	session := &discordgo.Session{}
	logger := slog.Default()
	client := NewClient("http://localhost:4200", session, logger)
	
	// Register multiple handlers
	handler1Called := false
	handler2Called := false
	
	handler1 := func(session *discordgo.Session, event *snitchv1.Event) error {
		handler1Called = true
		return nil
	}
	
	handler2 := func(session *discordgo.Session, event *snitchv1.Event) error {
		handler2Called = true
		return nil
	}
	
	client.RegisterHandler(snitchv1.EventType_EVENT_TYPE_REPORT_CREATED, handler1)
	client.RegisterHandler(snitchv1.EventType_EVENT_TYPE_REPORT_DELETED, handler2)
	
	// Test first handler
	event1 := &snitchv1.Event{Type: snitchv1.EventType_EVENT_TYPE_REPORT_CREATED}
	client.handleEvent(event1)
	
	if !handler1Called {
		t.Error("Handler 1 should have been called")
	}
	if handler2Called {
		t.Error("Handler 2 should not have been called")
	}
	
	// Reset and test second handler
	handler1Called = false
	handler2Called = false
	
	event2 := &snitchv1.Event{Type: snitchv1.EventType_EVENT_TYPE_REPORT_DELETED}
	client.handleEvent(event2)
	
	if handler1Called {
		t.Error("Handler 1 should not have been called")
	}
	if !handler2Called {
		t.Error("Handler 2 should have been called")
	}
}