package feature_wa_webhook

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Verify struct {
	Mode        string `form:"hub.mode"`
	VerifyToken string `form:"hub.verify_token"`
	Challenge   string `form:"hub.challenge"`
}

func (verify *Verify) Validate() []exception.InputException {
	verify.Mode = strings.TrimSpace(verify.Mode)
	verify.VerifyToken = strings.TrimSpace(verify.VerifyToken)
	verify.Challenge = strings.TrimSpace(verify.Challenge)
	errors := []exception.InputException{}
	if verify.Mode == "" {
		errors = append(errors, exception.NewInputException("hub.mode", "missing mode"))
	}
	if verify.VerifyToken == "" {
		errors = append(errors, exception.NewInputException("hub.verify_token", "missing verify token"))
	}
	if verify.Challenge == "" {
		errors = append(errors, exception.NewInputException("hub.challenge", "missing challenge"))
	}
	return errors
}

func (verify Verify) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
	if errors := verify.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}
	verifyToken := strings.TrimSpace(cfg.Default().WhatsApp.WebhookVerifyToken)
	if verify.Mode == "subscribe" && verifyToken != "" && verify.VerifyToken == verifyToken {
		return dto.NewSuccessResponse(verify.Challenge)
	}
	return dto.NewFailedResponse[string](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
}

func (Verify) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Verify WhatsApp webhook",
		"Verifies webhook subscription challenge from WhatsApp.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/receive",
		false,
		false,
		types.APITagWA,
		nil,
	)
}
