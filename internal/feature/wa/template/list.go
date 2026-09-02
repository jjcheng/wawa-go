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

type List struct {
	MetaWABAId string `form:"meta_waba_id" val:"required" description:"Meta WABA id"`
	Before     string `form:"before" description:"optional Meta pagination cursor"`
	After      string `form:"after" description:"optional Meta pagination cursor"`
	Limit      int    `form:"limit" description:"optional number of templates per page"`
}

func (list *List) Validate() []exception.InputException {
	list.MetaWABAId = strings.TrimSpace(list.MetaWABAId)
	list.Before = strings.TrimSpace(list.Before)
	list.After = strings.TrimSpace(list.After)
	if list.Limit == 0 {
		list.Limit = 10
	}
	inputErrors := []exception.InputException{}
	if list.MetaWABAId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	if list.Limit < 0 || list.Limit > 100 {
		inputErrors = append(inputErrors, exception.NewInputException("limit", "limit must be between 1 and 100"))
	}
	return inputErrors
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_wa.Template]] {
	if errors := list.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_wa.Template]](errors)
	}
	businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetValidBusinessAccount(ctx, user.Id, list.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusUnauthorized, "you are not authorized to access this business account")
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, businessAccount.MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	templates := make([]dto_wa.Template, 0)
	wabaTemplates, paging, err := dependencies.Whatsapp.ListTemplatesPage(ctx, list.MetaWABAId, list.Before, list.After, list.Limit, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	for index := range wabaTemplates {
		wabaTemplates[index].MetaWABAId = list.MetaWABAId
	}
	templates = append(templates, wabaTemplates...)
	additionalData := map[string]any{}
	if paging != nil {
		additionalData["previous"] = paging.Previous
		additionalData["next"] = paging.Next
	}
	response := dto.NewOffsetListResponse(templates, nil, &additionalData)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp templates",
		"Lists WhatsApp message templates available to the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this business account", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
		},
	)
}
