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
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
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

func (receive Receive) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[any] {
	if errors := receive.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		helper.WriteToFile(receive.RawBody, filepath.Join("files/wa", "receive.json"))
		if err := feature_wa_message.ProcessIncoming(ctx, receive.RawBody, &service.MessageQueueMessage{}, dependencies); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, err.Error())
		}
	} else {
		_, err := dependencies.MessageQueue.PublishJob("wa_incoming", receive.RawBody, 0, service.MessageQueuePriorityHighest)
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, "failed to queue raw webhook body")
		}
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
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
