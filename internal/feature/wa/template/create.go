package feature_wa_template

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
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
	phoneNumbers, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	authorized := false
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.MetaWABAId == create.MetaWABAId {
			authorized = true
			break
		}
	}
	if !authorized {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusUnauthorized, "you are not authorized to access this WABA")
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, phoneNumbers[0].MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	template, err := dependencies.Whatsapp.CreateTemplate(ctx, create.MetaWABAId, create.Payload(), businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewSuccessResponse(template)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp template",
		"Creates a WhatsApp message template for an authorized WABA.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
