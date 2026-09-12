package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ably/ably-go/ably"
	"github.com/jjcheng/wawa-go/internal/cfg"
)

type Ably struct {
	client *ably.REST
	logger *Logger
}

type AblyTokenRequest struct {
	TTL         int64  `json:"ttl"`
	Capability  string `json:"capability"`
	ClientID    string `json:"clientId"`
	Timestamp   int64  `json:"timestamp"`
	KeyName     string `json:"keyName"`
	Nonce       string `json:"nonce"`
	MAC         string `json:"mac"`
	ChannelName string `json:"channel_name"`
}

func NewAbly(logger *Logger) *Ably {
	apiKey := strings.TrimSpace(cfg.Default().Ably.APIKey)
	if apiKey == "" {
		panic("missing ably api key")
	}
	client, err := ably.NewREST(ably.WithKey(apiKey))
	if err != nil {
		panic("ably client error: " + err.Error())
	}
	return &Ably{client: client, logger: logger}
}

func (ably *Ably) Publish(eventName string, channelName string, payload any) error {
	return ably.publish(eventName, channelName, payload)
}

func (ablyService *Ably) CreateConversationTokenRequest(channelName string, clientID string) (*AblyTokenRequest, error) {
	if ablyService.client == nil {
		return nil, fmt.Errorf("Ably is not configured")
	}
	capability, err := json.Marshal(map[string][]string{channelName: {"subscribe", "presence"}})
	if err != nil {
		return nil, fmt.Errorf("failed to encode Ably token capability: %w", err)
	}
	request, err := ablyService.client.Auth.CreateTokenRequest(&ably.TokenParams{
		TTL:        int64((15 * time.Minute) / time.Millisecond),
		Capability: string(capability),
		ClientID:   clientID,
	})
	if err != nil {
		return nil, err
	}
	return &AblyTokenRequest{
		TTL:         request.TTL,
		Capability:  request.Capability,
		ClientID:    request.ClientID,
		Timestamp:   request.Timestamp,
		KeyName:     request.KeyName,
		Nonce:       request.Nonce,
		MAC:         request.MAC,
		ChannelName: channelName,
	}, nil
}

func (ablyService *Ably) publish(eventName string, channelName string, data any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ablyService.client.Channels.Get(channelName).Publish(ctx, eventName, data); err != nil {
		ablyService.logger.ErrorFunction(err, eventName, channelName)
		return err
	}
	return nil
}
