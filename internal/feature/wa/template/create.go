package feature_wa_template

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Create struct {
	MetaWABAId string           `json:"meta_waba_id" description:"Meta WABA ID"`
	Name       string           `json:"name" description:"template name"`
	Language   string           `json:"language" description:"template language code"`
	Category   string           `json:"category" description:"template category"`
	Components []map[string]any `json:"components" description:"template components"`
}

func (create *Create) Payload() map[string]any {
	return map[string]any{
		"name":       create.Name,
		"language":   create.Language,
		"category":   create.Category,
		"components": create.Components,
	}
}

func (create *Create) Validate() []exception.InputException {
	create.MetaWABAId = strings.TrimSpace(create.MetaWABAId)
	create.Name = strings.TrimSpace(create.Name)
	create.Language = strings.TrimSpace(create.Language)
	create.Category = strings.TrimSpace(create.Category)
	errors := []exception.InputException{}
	if create.MetaWABAId == "" {
		errors = append(errors, exception.NewInputException("meta_waba_id", "missing meta WABA id"))
	}
	if create.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing template name"))
	}
	if create.Language == "" {
		errors = append(errors, exception.NewInputException("language", "missing template language"))
	}
	if create.Category == "" {
		errors = append(errors, exception.NewInputException("category", "missing template category"))
	}
	if len(create.Components) == 0 {
		errors = append(errors, exception.NewInputException("components", "missing template components"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Template] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Template](errors)
	}

	phoneNumbers, ex := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if ex != nil {
		return dto.NewFailedResponse[*dto_wa.Template](ex.StatusCode, ex.Message)
	}
	authorized := false
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.MetaWABAId == create.MetaWABAId {
			authorized = true
			break
		}
	}
	if !authorized {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusForbidden, "you are not authorized to access this WABA")
	}

	template, err := dependencies.Whatsapp.CreateTemplate(ctx, create.MetaWABAId, create.Payload())
	if err != nil {
		dependencies.Logger.ErrorFunction(err, create.MetaWABAId, create.Name)
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, "failed to create WhatsApp template")
	}
	return dto.NewSuccessResponse(template)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp template",
		"Creates a WhatsApp message template for an authorized WABA.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/wa/v1/templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("failed to create WhatsApp template", http.StatusBadGateway)),
		},
	)
}
