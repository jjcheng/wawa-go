package service

import (
	"context"
	"encoding/json"
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
	logger          *Logger
	client          *eventbridge.Client
	eventBusName    string
	eventSourceName string
}

func NewEventBridge(logger *Logger) *EventBridge {
	config := cfg.Default().AliyunEventBridge
	eventBridge := &EventBridge{
		logger:          logger,
		eventBusName:    strings.TrimSpace(config.EventBusName),
		eventSourceName: strings.TrimSpace(config.EventSourceName),
	}
	if eventBridge.eventSourceName == "" {
		eventBridge.eventSourceName = "wawa-go"
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
	return eventBridge
}

// CreateEvent publishes a CloudEvent to the configured EventBridge bus.
// NOTE: EventBridge does not delay delivery based on CloudEvent.Time.
// The event is published immediately; the caller should only use this value for metadata.
func (eventBridge *EventBridge) CreateEvent(ctx context.Context, name string, typ string, subject string, sendAt time.Time, userData map[string]*string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if sendAt.IsZero() {
		sendAt = time.Now().UTC()
	}
	if userData == nil {
		userData = map[string]*string{}
	}
	if _, ok := userData["scheduled_at"]; !ok {
		userData["scheduled_at"] = helper.ConvertToPointer(sendAt.UTC().Format(time.RFC3339))
	}
	payload, err := json.Marshal(userData)
	if err != nil {
		return "", fmt.Errorf("marshal EventBridge payload: %w", err)
	}
	event := &eventbridge.CloudEvent{
		Id:              helper.ConvertToPointer(name),
		Source:          helper.ConvertToPointer(eventBridge.eventSourceName),
		Type:            helper.ConvertToPointer(typ),
		Datacontenttype: helper.ConvertToPointer("application/json;charset=utf-8"),
		Subject:         helper.ConvertToPointer(subject),
		Time:            helper.ConvertToPointer(time.Now().UTC().Format(time.RFC3339)),
		Data:            payload,
		Extensions: map[string]any{
			"aliyuneventbusname": eventBridge.eventBusName,
			"scheduled_at":       sendAt.UTC().Format(time.RFC3339),
		},
	}
	response, err := eventBridge.client.PutEvents([]*eventbridge.CloudEvent{event})
	if err != nil {
		return "", fmt.Errorf("publish EventBridge event: %w", err)
	}
	if response == nil || response.RequestId == nil {
		return "", errors.New("EventBridge returned no publish result")
	}
	if response.FailedEntryCount != nil && *response.FailedEntryCount > 0 {
		return "", fmt.Errorf("EventBridge publish had %d failed entries", *response.FailedEntryCount)
	}
	return *response.RequestId, nil
}

// DeleteEvent is a compatibility no-op because the publish flow does not create EventBridge sources.
func (eventBridge *EventBridge) DeleteEvent(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if eventBridge == nil || eventBridge.client == nil {
		return nil
	}
	if strings.TrimSpace(name) == "" {
		return nil
	}
	return nil
}

// DeleteSchedule is an alias for DeleteEvent.
func (eventBridge *EventBridge) DeleteSchedule(ctx context.Context, name string) error {
	return eventBridge.DeleteEvent(ctx, name)
}
