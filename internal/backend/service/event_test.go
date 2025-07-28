package service

import (
	"database/sql"
	"testing"
	"time"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	_ "github.com/tursodatabase/go-libsql"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEventService_PublishEvent(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("libsql", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Error("Failed to close database", "error", err)
		}
	}()

	service := NewEventService(db)

	// Test event publishing to subscribers with group filtering
	eventChan := make(chan *snitchv1.Event, 10)

	// Create subscriber for group "test-group"
	sub := &subscriber{
		eventChan: eventChan,
		groupID:   "test-group",
	}

	service.mu.Lock()
	service.subscribers[sub] = true
	service.mu.Unlock()

	// Test event for same group should be delivered
	testEvent := &snitchv1.Event{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		ServerId:  "test-server",
		GroupId:   "test-group",
		Timestamp: timestamppb.Now(),
	}

	err = service.PublishEvent(testEvent)
	if err != nil {
		t.Errorf("PublishEvent failed: %v", err)
	}

	select {
	case receivedEvent := <-eventChan:
		if receivedEvent.Type != snitchv1.EventType_EVENT_TYPE_REPORT_CREATED {
			t.Errorf("Expected EVENT_TYPE_REPORT_CREATED, got %v", receivedEvent.Type)
		}
		if receivedEvent.GroupId != "test-group" {
			t.Errorf("Expected group_id 'test-group', got %v", receivedEvent.GroupId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event not received")
	}
}

func TestEventService_GroupFiltering(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("libsql", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Error("Failed to close database", "error", err)
		}
	}()

	service := NewEventService(db)

	// Create subscribers for different groups
	group1Chan := make(chan *snitchv1.Event, 10)
	group2Chan := make(chan *snitchv1.Event, 10)

	sub1 := &subscriber{
		eventChan: group1Chan,
		groupID:   "group-1",
	}

	sub2 := &subscriber{
		eventChan: group2Chan,
		groupID:   "group-2",
	}

	service.mu.Lock()
	service.subscribers[sub1] = true
	service.subscribers[sub2] = true
	service.mu.Unlock()

	// Send event to group-1
	testEvent := &snitchv1.Event{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		ServerId:  "test-server",
		GroupId:   "group-1",
		Timestamp: timestamppb.Now(),
	}

	err = service.PublishEvent(testEvent)
	if err != nil {
		t.Errorf("PublishEvent failed: %v", err)
	}

	// group-1 should receive the event
	select {
	case receivedEvent := <-group1Chan:
		if receivedEvent.GroupId != "group-1" {
			t.Errorf("Expected group_id 'group-1', got %v", receivedEvent.GroupId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event not received by group-1 subscriber")
	}

	// group-2 should NOT receive the event
	select {
	case <-group2Chan:
		t.Error("group-2 should not have received event for group-1")
	case <-time.After(50 * time.Millisecond):
		// This is expected - no event should be received
	}
}

func TestEventService_ChannelFullHandling(t *testing.T) {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("libsql", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Error("Failed to close database", "error", err)
		}
	}()

	service := NewEventService(db)

	// Test that full channels don't block publishing
	testEvent := &snitchv1.Event{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		GroupId:   "test-group",
		Timestamp: timestamppb.Now(),
	}

	// Create a full channel
	fullChan := make(chan *snitchv1.Event)
	sub := &subscriber{
		eventChan: fullChan,
		groupID:   "test-group",
	}

	service.mu.Lock()
	service.subscribers[sub] = true
	service.mu.Unlock()

	// This should not block and should return an error indicating dropped events
	err = service.PublishEvent(testEvent)
	if err == nil {
		t.Error("Expected error for dropped events")
	}
}
