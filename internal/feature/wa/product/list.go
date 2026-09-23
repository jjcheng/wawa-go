package feature_wa_product

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

type List struct {
	CatalogId string `uri:"id" val:"required" description:"id of the Meta commerce catalog"`
	Before    string `form:"before" description:"optional Meta pagination cursor for the previous page"`
	After     string `form:"after" description:"optional Meta pagination cursor for the next page"`
	Limit     int    `form:"limit" description:"optional number of products per page"`
}

func (list *List) Validate() []exception.InputException {
	list.CatalogId = strings.TrimSpace(list.CatalogId)
	list.Before = strings.TrimSpace(list.Before)
	list.After = strings.TrimSpace(list.After)
	if list.Limit == 0 {
		list.Limit = 25
	}
	inputErrors := []exception.InputException{}
	if list.CatalogId == "" {
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

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[service.WhatsAppProduct]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProduct]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProduct]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProduct]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[service.WhatsAppProduct]](inputErrors)
	}
	products, paging, err := dependencies.Whatsapp.ListProductsByCatalogId(ctx, list.CatalogId, list.Before, list.After, list.Limit, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[service.WhatsAppProduct]](http.StatusBadGateway, err.Error(), err)
	}
	additionalData := map[string]any{}
	if paging != nil {
		if paging.Cursors != nil {
			additionalData["before"] = paging.Cursors.Before
			additionalData["after"] = paging.Cursors.After
		}
		additionalData["previous"] = paging.Previous
		if len(products) == list.Limit {
			additionalData["next"] = paging.Next
		}
	}
	response := dto.NewOffsetListResponse(products, nil, &additionalData)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp catalog products",
		"Lists products in a Meta commerce catalog.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/catalogs/:id/products",
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
