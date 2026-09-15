package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/eventbridge-sdk/eventbridge"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
)

// EventBridge publishes CloudEvents to Alibaba Cloud EventBridge.
// It is disabled when its credentials, endpoint, or event bus are not configured.
type EventBridge struct {
	logger       *Logger
	scheduler    *eventbridge.Client
	eventBusName string
}

func NewEventBridge(logger *Logger) *EventBridge {
	config := cfg.Default().AliyunEventBridge
	eventBridge := &EventBridge{
		logger:       logger,
		eventBusName: strings.TrimSpace(config.EventBusName),
	}
	if strings.TrimSpace(config.Endpoint) == "" || strings.TrimSpace(config.AccessKeyID) == "" || strings.TrimSpace(config.AccessKeySecret) == "" || eventBridge.eventBusName == "" {
		panic("invalid EventBridge configuration")
	}
	scheduler, err := eventbridge.NewClient(new(eventbridge.Config).
		SetAccessKeyId(strings.TrimSpace(config.AccessKeyID)).
		SetAccessKeySecret(strings.TrimSpace(config.AccessKeySecret)).
		SetEndpoint(strings.TrimSpace(config.Endpoint)))
	if err != nil {
		panic(err)
	}
	eventBridge.scheduler = scheduler
	return eventBridge
}

// CreateEvent creates a one-time EventBridge scheduled event source.
// Configure an EventBridge rule to route scheduled events to Function Compute.
func (eventBridge *EventBridge) CreateEvent(ctx context.Context, name string, sendAt time.Time) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if eventBridge == nil || eventBridge.scheduler == nil {
		return "", errors.New("Alibaba EventBridge scheduler is not configured")
	}
	if sendAt.IsZero() || !sendAt.After(time.Now().UTC()) {
		return "", errors.New("scheduled time must be in the future")
	}
	schedule := fmt.Sprintf("at(%s)", sendAt.UTC().Format(time.RFC3339))
	response, err := eventBridge.scheduler.CreateEventSource(&eventbridge.CreateEventSourceRequest{
		Description:     helper.ConvertToPointer(fmt.Sprintf("Scheduled event: %s", name)),
		EventBusName:    helper.ConvertToPointer(eventBridge.eventBusName),
		EventSourceName: helper.ConvertToPointer(name),
		SourceScheduledEventParameters: &eventbridge.SourceScheduledEventParameters{
			Schedule: helper.ConvertToPointer(schedule),
			TimeZone: helper.ConvertToPointer("UTC"),
		},
	})
	if err != nil {
		return "", fmt.Errorf("create EventBridge schedule: %w", err)
	}
	if response == nil || response.EventSourceARN == nil {
		return "", errors.New("EventBridge returned no scheduled event source")
	}
	return *response.EventSourceARN, nil
}

// DeleteEvent removes a scheduled event source by name.
func (eventBridge *EventBridge) DeleteEvent(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if eventBridge == nil || eventBridge.scheduler == nil {
		return errors.New("Alibaba EventBridge scheduler is not configured")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("scheduled event source name is required")
	}
	_, err := eventBridge.scheduler.DeleteEventSource(&eventbridge.DeleteEventSourceRequest{
		EventBusName:    helper.ConvertToPointer(eventBridge.eventBusName),
		EventSourceName: helper.ConvertToPointer(name),
	})
	if err != nil {
		return fmt.Errorf("delete EventBridge schedule: %w", err)
	}
	return nil
}

// DeleteSchedule is an alias for DeleteEvent.
func (eventBridge *EventBridge) DeleteSchedule(ctx context.Context, name string) error {
	return eventBridge.DeleteEvent(ctx, name)
}
