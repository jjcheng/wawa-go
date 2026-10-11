package feature_wa_webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
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
	errors := []exception.InputException{}
	if strings.TrimSpace(receive.RawBody) == "" {
		errors = append(errors, exception.NewInputException("body", "missing raw body"))
	}
	return errors
}

func (receive Receive) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[any] {
	if errors := receive.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	// for development, process it straightaway; for staging/production, push rawBody to SMQ
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		helper.WriteToFile(receive.RawBody, filepath.Join("files/receive", fmt.Sprintf("receive_%s.json", uuid.NewString())))
		waIncoming, err := helper.DeserializeJSON[dto_wa.Incoming](receive.RawBody)
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, err.Error(), err)
		}
		if err := feature_wa_message.ProcessIncoming(context.Background(), *waIncoming, dependencies); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, err.Error(), err)
		}
	} else {
		_, err := dependencies.MessageQueue.PublishJob("handle_wa_incoming", receive.RawBody, 0, service.MessageQueuePriorityHighest)
		// Meta will retry sending if failed here
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, "failed to queue raw webhook body", err)
		}
		// TODO: temporarily forward the raw message to https://chooser-elude-author.ngrok-free.dev
		if err := receive.forwardRawBody(ctx); err != nil {
			dependencies.Logger.ErrorFunction(err)
		}
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (receive Receive) forwardRawBody(ctx context.Context) error {
	forwardContext, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(forwardContext, http.MethodPost,
		"https://chooser-elude-author.ngrok-free.dev/v1/wa/receive", strings.NewReader(receive.RawBody))
	if err != nil {
		return fmt.Errorf("failed to create forwarded webhook request: %w", err)
	}
	appSecret := strings.TrimSpace(cfg.Default().WhatsApp.AppSecret)
	if appSecret == "" {
		return fmt.Errorf("cannot forward webhook without WhatsApp app secret")
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write([]byte(receive.RawBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	client := &http.Client{
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to forward webhook: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("forwarded webhook returned HTTP %d", response.StatusCode)
	}
	return nil
}

func (Receive) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"To receive incoming WhatsApp events",
		"Receive incoming WhatsApp webhook events and pushes them into the queue.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/receive",
		false,
		false,
		types.APITagWA,
		nil,
	)
}
