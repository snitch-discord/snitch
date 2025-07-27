package service

import (
	"testing"
	"time"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
)

func TestEventService_PublishEvent(t *testing.T) {
	service := NewEventService()
	
	// Test event publishing to subscribers
	eventChan := make(chan *snitchv1.Event, 10)
	
	service.mu.Lock()
	service.subscribers[eventChan] = true
	service.mu.Unlock()
	
	testEvent := &snitchv1.Event{
		Type:     snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		ServerId: "test-server",
	}
	
	service.PublishEvent(testEvent)
	
	select {
	case receivedEvent := <-eventChan:
		if receivedEvent.Type != snitchv1.EventType_EVENT_TYPE_REPORT_CREATED {
			t.Errorf("Expected EVENT_TYPE_REPORT_CREATED, got %v", receivedEvent.Type)
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("Event not received")
	}
}

func TestEventService_ChannelFullHandling(t *testing.T) {
	service := NewEventService()
	
	// Test that full channels don't block publishing
	testEvent := &snitchv1.Event{
		Type: snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
	}
	
	// Create a full channel
	fullChan := make(chan *snitchv1.Event)
	service.mu.Lock()
	service.subscribers[fullChan] = true
	service.mu.Unlock()
	
	// This should not block
	service.PublishEvent(testEvent)
}