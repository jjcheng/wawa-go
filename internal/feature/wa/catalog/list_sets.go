package feature_wa_catalog

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type ListSets struct {
	CatalogID string `uri:"id" val:"required" description:"id of the Meta commerce catalog"`
	Before    string `form:"before" description:"optional Meta pagination cursor for the previous page"`
	After     string `form:"after" description:"optional Meta pagination cursor for the next page"`
	Limit     int    `form:"limit" description:"optional number of product sets per page"`
}

func (list *ListSets) Validate() []exception.InputException {
	list.CatalogID = strings.TrimSpace(list.CatalogID)
	list.Before = strings.TrimSpace(list.Before)
	list.After = strings.TrimSpace(list.After)
	if list.Limit == 0 {
		list.Limit = 100
	}
	inputErrors := []exception.InputException{}
	if list.CatalogID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing catalog ID"))
	}
	if list.Before != "" && list.After != "" {
		inputErrors = append(inputErrors, exception.NewInputException("before", "before and after cannot both be provided"))
	}
	if list.Limit < 1 || list.Limit > 100 {
		inputErrors = append(inputErrors, exception.NewInputException("limit", "limit must be between 1 and 100"))
	}
	return inputErrors
}

func (list ListSets) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[service.WhatsAppProductSet]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProductSet]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProductSet]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProductSet]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[service.WhatsAppProductSet]](inputErrors)
	}
	sets, paging, err := dependencies.Whatsapp.ListProductSets(ctx, list.CatalogID, list.Before, list.After, list.Limit, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProductSet]](http.StatusBadGateway, err.Error(), err)
	}
	additionalData := map[string]any{}
	if paging != nil {
		if paging.Cursors != nil {
			additionalData["before"] = paging.Cursors.Before
			additionalData["after"] = paging.Cursors.After
		}
		additionalData["previous"] = paging.Previous
		if len(sets) == list.Limit {
			additionalData["next"] = paging.Next
		}
	}
	response := dto.NewOffsetListResponse(sets, nil, &additionalData)
	return dto.NewSuccessResponse(&response)
}

func (ListSets) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp catalog product sets",
		"Lists product sets in a Meta commerce catalog.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/catalogs/:id/sets",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
