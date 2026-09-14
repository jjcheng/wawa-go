package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/models"
	eventbridgeapi "github.com/alibabacloud-go/eventbridge-20200401/client"
	"github.com/alibabacloud-go/eventbridge-sdk/eventbridge"
	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
)

const eventBridgeBusNameExtension = "aliyuneventbusname"

type eventBridgeClient interface {
	PutEvents(eventList []*eventbridge.CloudEvent) (*eventbridge.PutEventsResponse, error)
}

// EventBridge publishes CloudEvents to Alibaba Cloud EventBridge.
// It is disabled when its credentials, endpoint, or event bus are not configured.
type EventBridge struct {
	logger       *Logger
	client       eventBridgeClient
	scheduler    *eventbridgeapi.Client
	eventBusName string
	source       string
}

func NewEventBridge(logger *Logger) *EventBridge {
	config := cfg.Default().AliyunEventBridge
	eventBridge := &EventBridge{
		logger:       logger,
		eventBusName: strings.TrimSpace(config.EventBusName),
		source:       strings.TrimSpace(config.Source),
	}
	if strings.TrimSpace(config.Endpoint) == "" || strings.TrimSpace(config.AccessKeyID) == "" || strings.TrimSpace(config.AccessKeySecret) == "" || eventBridge.eventBusName == "" {
		panic("invalid EventBridge configuration")
	}
	client, err := eventbridge.NewClient(new(eventbridge.Config).
		SetAccessKeyId(strings.TrimSpace(config.AccessKeyID)).
		SetAccessKeySecret(strings.TrimSpace(config.AccessKeySecret)).
		SetEndpoint(strings.TrimSpace(config.Endpoint)))
	if err != nil {
		panic(err)
	}
	eventBridge.client = client
	scheduler, err := eventbridgeapi.NewClient(new(openapi.Config).
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
	response, err := eventBridge.scheduler.CreateEventSource(&eventbridgeapi.CreateEventSourceRequest{
		Description:     helper.ConvertToPointer(fmt.Sprintf("Scheduled event: %s", name)),
		EventBusName:    helper.ConvertToPointer(eventBridge.eventBusName),
		EventSourceName: helper.ConvertToPointer(name),
		SourceScheduledEventParameters: &eventbridgeapi.CreateEventSourceRequestSourceScheduledEventParameters{
			Schedule: helper.ConvertToPointer(schedule),
			TimeZone: helper.ConvertToPointer("UTC"),
		},
	})
	if err != nil {
		return "", fmt.Errorf("create EventBridge schedule: %w", err)
	}
	if response == nil || response.Body == nil || response.Body.Data == nil || response.Body.Data.EventSourceARN == nil {
		return "", errors.New("EventBridge returned no scheduled event source")
	}
	return *response.Body.Data.EventSourceARN, nil
}

// DeleveEvent removes a scheduled event source by name.
func (eventBridge *EventBridge) DeleveEvent(ctx context.Context, name string) error {
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
	_, err := eventBridge.scheduler.DeleteEventSource(&eventbridgeapi.DeleteEventSourceRequest{
		EventSourceName: helper.ConvertToPointer(name),
	})
	if err != nil {
		return fmt.Errorf("delete EventBridge schedule: %w", err)
	}
	return nil
}

// Publish marshals payload as JSON and sends it as a CloudEvent.
func (eventBridge *EventBridge) Publish(ctx context.Context, eventType string, subject string, payload any) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if eventBridge == nil || eventBridge.client == nil {
		return "", errors.New("Alibaba EventBridge is not configured")
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return "", errors.New("event type is required")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal EventBridge event payload: %w", err)
	}
	event := new(eventbridge.CloudEvent).
		SetId(uuid.NewString()).
		SetSource(eventBridge.source).
		SetSpecversion("1.0").
		SetType(eventType).
		SetSubject(strings.TrimSpace(subject)).
		SetTime(time.Now().UTC().Format(time.RFC3339Nano)).
		SetDatacontenttype("application/json").
		SetData(data).
		SetExtensions(map[string]interface{}{eventBridgeBusNameExtension: eventBridge.eventBusName})
	response, err := eventBridge.client.PutEvents([]*eventbridge.CloudEvent{event})
	if err != nil {
		return "", fmt.Errorf("publish EventBridge event: %w", err)
	}
	if response == nil || len(response.EntryList) != 1 || response.EntryList[0] == nil {
		return "", errors.New("EventBridge returned no result for published event")
	}
	entry := response.EntryList[0]
	if entry.ErrorCode != nil || entry.ErrorMessage != nil || response.FailedEntryCount != nil && *response.FailedEntryCount > 0 {
		return "", fmt.Errorf("EventBridge rejected event: code=%s message=%s", valueOrEmpty(entry.ErrorCode), valueOrEmpty(entry.ErrorMessage))
	}
	if entry.EventId == nil || strings.TrimSpace(*entry.EventId) == "" {
		return "", errors.New("EventBridge returned an empty event ID")
	}
	eventID := strings.TrimSpace(*entry.EventId)
	if eventBridge.logger != nil {
		eventBridge.logger.Debugf("EventBridge event published: bus=%s event_id=%s type=%s", eventBridge.eventBusName, eventID, eventType)
	}
	return eventID, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
