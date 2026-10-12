package feature_wa_message

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type MarkRead struct {
	CustomerId int32 `form:"customer_id" val:"required" description:"all messages with this customer will be read"`
}

func (markRead *MarkRead) Validate() []exception.InputException {
	if markRead.CustomerId <= 0 {
		return []exception.InputException{exception.NewInputException("customer_id", "missing customer id")}
	}
	return nil
}

func (markRead MarkRead) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolioAccessToken == "" {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := markRead.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, markRead.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "customer not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// check customer belong to this user
	if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == customer.PhoneNumberId
	}) {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if err := dependencies.UnitOfWork.WAMessageRepository().SetReadByCustomerId(ctx, markRead.CustomerId, true); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (MarkRead) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Mark WhatsApp message as read",
		"Mark a WhatsApp message as read by user.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/messages/mark-read",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("message not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
