package feature_broadcast_recipient

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type List struct {
	Name        string                `form:"name" description:"name of the customer to filter"`
	BroadcastId int32                 `form:"broadcast_id" val:"required" description:"id of the broadcast"`
	Status      types.WAMessageStatus `form:"status" description:"optional recipient status filter, use message status each recipient is tied to a message"`
	Page        int                   `form:"page" description:"page number from 1"`
	PageSize    int                   `form:"page_size" description:"number per page"`
}

func (list *List) Validate() []exception.InputException {
	list.Name = strings.TrimSpace(list.Name)
	if list.BroadcastId <= 0 {
		return []exception.InputException{exception.NewInputException("broadcast_id", "invalid broadcast id")}
	}
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 25
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_customer.BroadcastRecipient]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.BroadcastRecipient]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_customer.BroadcastRecipient]](inputErrors)
	}
	broadcast, err := dependencies.UnitOfWork.BroadcastRepository().GetById(ctx, list.BroadcastId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_customer.BroadcastRecipient]](http.StatusNotFound, "broadcast not found", nil)
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.BroadcastRecipient]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if broadcast.UserId != user.Id {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.BroadcastRecipient]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	recipients, totalCount, totalPages, err := dependencies.UnitOfWork.BroadcastRecipientRepository().ListByBroadcastId(ctx, list.BroadcastId, list.Name, list.Status, false, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.BroadcastRecipient]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	items := make([]dto_customer.BroadcastRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		items = append(items, dto_customer.NewBroadcastRecipient(recipient))
	}
	response := dto.NewPagedListResponse(items, totalPages, totalCount)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List broadcast recipients",
		"Lists recipients belonging to an authenticated user's broadcast.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/broadcasts/recipients",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
