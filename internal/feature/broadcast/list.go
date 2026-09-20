package feature_broadcast

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
	Name     string                `form:"name" description:"filter broadcasts by name"`
	Status   types.BroadcastStatus `form:"status" description:"filter broadcasts by status"`
	Page     int                   `form:"page" description:"page number from 1"`
	PageSize int                   `form:"page_size" description:"number per page"`
}

func (list *List) Validate() []exception.InputException {
	list.Name = strings.TrimSpace(list.Name)
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 25
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_customer.Broadcast]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.Broadcast]](http.StatusForbidden, "you are not authenticated")
	}
	if errors := list.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_customer.Broadcast]](errors)
	}
	broadcasts, err := dependencies.UnitOfWork.BroadcastRepository().ListByUserId(ctx, user.Id, list.Name, list.Status, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.Broadcast]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	items := make([]dto_customer.Broadcast, 0, len(broadcasts.Items))
	for _, broadcast := range broadcasts.Items {
		items = append(items, dto_customer.NewBroadcast(broadcast))
	}
	response := dto.NewPagedListResponse(items, broadcasts.NumberOfPages, broadcasts.NumberOfItems)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp broadcasts",
		"Lists broadcasts belonging to the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/broadcasts",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
