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

type List struct {
	NameOrContent string                       `form:"name_or_content" description:"optional template name or content search"`
	Category      types.WATemplateCategory     `form:"category" description:"optional template category filter"`
	Language      string                       `form:"language" description:"optional template language filter"`
	Status        types.WATemplateStatus       `form:"status" description:"optional template status filter"`
	QualityScore  types.WATemplateQualityScore `form:"quality_score" description:"optional template quality score filter"`
	Before        string                       `form:"before" description:"optional Meta pagination cursor for the previous page"`
	After         string                       `form:"after" description:"optional Meta pagination cursor for the next page"`
	Limit         int                          `form:"limit" description:"optional number of templates per page"`
}

func (list *List) Validate() []exception.InputException {
	list.NameOrContent = strings.TrimSpace(list.NameOrContent)
	list.Language = strings.TrimSpace(list.Language)
	list.Before = strings.TrimSpace(list.Before)
	list.After = strings.TrimSpace(list.After)
	if list.Limit == 0 {
		list.Limit = 10
	}
	inputErrors := []exception.InputException{}
	if list.Before != "" && list.After != "" {
		inputErrors = append(inputErrors, exception.NewInputException("before", "before and after cannot both be provided"))
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
	if user.WA == nil || user.WA.BusinessAccount == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusUnauthorized, "you are not authorized to access this business account")
	}
	metaWABAId := user.WA.BusinessAccount.WABAId
	wabaTemplates, paging, err := dependencies.Whatsapp.ListTemplates(ctx, metaWABAId, list.NameOrContent, list.Category, list.Language, list.Status, list.QualityScore, list.Before, list.After, list.Limit, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Template]](http.StatusBadGateway, err.Error())
	}
	templates := append(make([]dto_wa.Template, 0, len(wabaTemplates)), wabaTemplates...)
	additionalData := map[string]any{}
	if paging != nil {
		if paging.Cursors != nil {
			additionalData["before"] = paging.Cursors.Before
			additionalData["after"] = paging.Cursors.After
		}
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
