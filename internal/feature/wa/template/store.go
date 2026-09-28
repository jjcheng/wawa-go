package feature_wa_template

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Store struct {
	Id         string                     `json:"id" description:"id of the template, if updating"`
	Name       string                     `json:"name" val:"required" description:"name of the template, will add api_ prefix automatically"`
	Category   types.WATemplateCategory   `json:"category" val:"required" description:"category of the template"`
	Language   string                     `json:"language" val:"required" description:"language code of the template"`
	Components []dto_wa.TemplateComponent `json:"components,omitempty" description:"components of the template"`
}

func (store *Store) Validate() []exception.InputException {
	store.Id = strings.TrimSpace(store.Id)
	store.Name = strings.TrimSpace(store.Name)
	store.Language = strings.TrimSpace(store.Language)
	errors := []exception.InputException{}
	if store.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing template name"))
	}
	if store.Language == "" {
		errors = append(errors, exception.NewInputException("language", "missing template language"))
	}
	if store.Category == "" {
		errors = append(errors, exception.NewInputException("category", "missing template category"))
	}
	if len(store.Components) == 0 {
		errors = append(errors, exception.NewInputException("components", "missing template components"))
	}
	return errors
}

func (store Store) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Template] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := store.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Template](errors)
	}
	if store.Id != "" {
		existing, err := dependencies.Whatsapp.GetTemplate(ctx, store.Id, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, err.Error(), err)
		}
		if !strings.HasPrefix(existing.Name, "api_") {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadRequest, "only template created by API can be edited", nil)
		}
		payload := map[string]any{
			"category":   store.Category,
			"components": store.Components,
		}
		if err := dependencies.Whatsapp.UpdateTemplate(ctx, store.Id, payload, user.WA.BusinessPortfolioAccessToken); err != nil {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, err.Error(), err)
		}
		template, err := dependencies.Whatsapp.GetTemplate(ctx, store.Id, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, err.Error(), err)
		}
		return dto.NewSuccessResponse(template)
	}
	// we append api_ as the prefix of the template name to identify it's created from the api
	store.Name = "api_" + store.Name
	payload := map[string]any{
		"name":       store.Name,
		"language":   store.Language,
		"category":   store.Category,
		"components": store.Components,
	}
	template, err := dependencies.Whatsapp.CreateTemplate(ctx, user.WA.BusinessAccount.WABAId, payload, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, err.Error(), err)
	}
	// insert cache for template_id_user_id, expires in 1 hour
	cacheKey := helper.GetTemplateStatusChangeCacheKey(template.ID)
	cacheValue := fmt.Sprint(user.Id)
	// expires in 1 hours
	err = dependencies.Cache.Set(ctx, cacheKey, cacheValue, time.Hour*1)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, store.Id)
	}
	return dto.NewSuccessResponse(template)
}

func (Store) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create or update WhatsApp template",
		"Creates a WhatsApp message template, or updates its category and components when an ID is supplied. Name and language cannot be changed.",
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
