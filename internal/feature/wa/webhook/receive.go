package feature_wa_webhook

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Receive struct {
	RawBody string `json:"-"`
}

func (receive *Receive) Validate() []exception.InputException {
	receive.RawBody = strings.TrimSpace(receive.RawBody)
	errors := []exception.InputException{}
	if receive.RawBody == "" {
		errors = append(errors, exception.NewInputException("body", "missing raw body"))
	}
	return errors
}

func (receive Receive) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[map[string]any] {
	if errors := receive.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[map[string]any](errors)
	}
	// save raw body to file receive.json
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		helper.WriteToFile(receive.RawBody, filepath.Join("files/wa", "receive.json"))
	}
	messageID, err := dependencies.MessageQueue.PublishMessage(receive.RawBody, 0, service.MessageQueuePriorityHighest)
	if err != nil {
		return dto.NewFailedResponse[map[string]any](http.StatusInternalServerError, "failed to queue raw webhook body")
	}
	return dto.NewSuccessResponse(map[string]any{"message_id": messageID})
}

func (Receive) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Webhook to receive incoming WhatsApp events",
		"Receives incoming WhatsApp webhook events and pushes incoming messages into the queue.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/receive",
		false,
		false,
		types.APITagWA,
		nil,
	)
}
