package service

import (
	"testing"
	"time"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const TEST_GROUP_ID = "test-group-id"
const TEST_SERVER_ID = "test-server-id"

func TestEventService_PublishEvent(t *testing.T) {
	// PublishEvent doesn't use dbClient, so we can pass nil
	service := NewEventService(nil, "test-jwt-secret")

	// Test event publishing to subscribers with group filtering
	eventChan := make(chan *snitchv1.SubscribeResponse, 10)

	// Create subscriber for group "test-group"
	sub := &subscriber{
		eventChan: eventChan,
		groupID:   TEST_GROUP_ID,
	}

	service.mu.Lock()
	service.subscribers[sub] = true
	service.mu.Unlock()

	// Test event for same group should be delivered
	testEvent := &snitchv1.SubscribeResponse{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		ServerId:  TEST_SERVER_ID,
		GroupId:   TEST_GROUP_ID,
		Timestamp: timestamppb.Now(),
	}

	err := service.PublishEvent(t.Context(), testEvent)
	if err != nil {
		t.Errorf("PublishEvent failed: %v", err)
	}

	select {
	case receivedEvent := <-eventChan:
		if receivedEvent.Type != snitchv1.EventType_EVENT_TYPE_REPORT_CREATED {
			t.Errorf("Expected '%s', got %v", snitchv1.EventType_EVENT_TYPE_REPORT_CREATED, receivedEvent.Type)
		}
		if receivedEvent.GroupId != TEST_GROUP_ID {
			t.Errorf("Expected group_id '%s', got %v", TEST_GROUP_ID, receivedEvent.GroupId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event not received")
	}
}

func TestEventService_GroupFiltering(t *testing.T) {
	group1ID := "group-1"
	group2ID := "group-2"

	// PublishEvent doesn't use dbClient, so we can pass nil
	service := NewEventService(nil, "test-jwt-secret")

	// Create subscribers for different groups
	group1Chan := make(chan *snitchv1.SubscribeResponse, 10)
	group2Chan := make(chan *snitchv1.SubscribeResponse, 10)

	sub1 := &subscriber{
		eventChan: group1Chan,
		groupID:   group1ID,
	}

	sub2 := &subscriber{
		eventChan: group2Chan,
		groupID:   group2ID,
	}

	service.mu.Lock()
	service.subscribers[sub1] = true
	service.subscribers[sub2] = true
	service.mu.Unlock()

	// Send event to group-1
	testEvent := &snitchv1.SubscribeResponse{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		ServerId:  TEST_SERVER_ID,
		GroupId:   group1ID,
		Timestamp: timestamppb.Now(),
	}

	err := service.PublishEvent(t.Context(), testEvent)
	if err != nil {
		t.Errorf("PublishEvent failed: %v", err)
	}

	// group-1 should receive the event
	select {
	case receivedEvent := <-group1Chan:
		if receivedEvent.GroupId != group1ID {
			t.Errorf("Expected group_id '%s', got %v", group1ID, receivedEvent.GroupId)
		}
	case <-time.After(100 * time.Millisecond):
		t.Errorf("Event not received by '%s' subscriber", group1ID)
	}

	// group-2 should NOT receive the event
	select {
	case <-group2Chan:
		t.Errorf("%s should not have received event for %s", group2ID, group1ID)
	case <-time.After(50 * time.Millisecond):
		// This is expected - no event should be received
	}
}

func TestEventService_ChannelFullHandling(t *testing.T) {
	// PublishEvent doesn't use dbClient, so we can pass nil
	service := NewEventService(nil, "test-jwt-secret")

	// Test that full channels don't block publishing
	testEvent := &snitchv1.SubscribeResponse{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		GroupId:   TEST_GROUP_ID,
		Timestamp: timestamppb.Now(),
	}

	// Create a full channel
	fullChan := make(chan *snitchv1.SubscribeResponse)
	sub := &subscriber{
		eventChan: fullChan,
		groupID:   TEST_GROUP_ID,
	}

	service.mu.Lock()
	service.subscribers[sub] = true
	service.mu.Unlock()

	// This should not block and should return an error indicating dropped events
	err := service.PublishEvent(t.Context(), testEvent)
	if err == nil {
		t.Error("Expected error for dropped events")
	}
}
