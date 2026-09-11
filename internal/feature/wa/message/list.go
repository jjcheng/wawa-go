package feature_wa_message

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
	PhoneNumberId      string `form:"phone_number_id" val:"required" description:"user's phone number id"`
	CustomerWAId       string `form:"customer_wa_id" description:"country code + phone number, all numbers, must present if customer_meta_user_id is empty" example:"6590909900"`
	CustomerMetaUserId string `form:"customer_meta_user_id" description:"user id from Meta, must present if customer WA Id is empty"`
	Page               int    `form:"page" description:"page number from 1"`
	PageSize           int    `form:"page_size" description:"page size, default 50"`
}

func (list *List) Validate() []exception.InputException {
	list.CustomerWAId = strings.TrimSpace(list.CustomerWAId)
	list.CustomerMetaUserId = strings.TrimSpace(list.CustomerMetaUserId)
	if list.Page == 0 {
		list.Page = 1
	}
	if list.PageSize == 0 {
		list.PageSize = 50
	}
	inputErrors := []exception.InputException{}
	if list.PhoneNumberId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "missing phone number ID"))
	}
	if list.CustomerWAId == "" && list.CustomerMetaUserId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("customer_wa_id", "customerWA Id or Meta user ID is required"))
	}
	if list.Page < 1 {
		inputErrors = append(inputErrors, exception.NewInputException("page", "page must be at least 1"))
	}
	if list.PageSize < 1 || list.PageSize > 100 {
		inputErrors = append(inputErrors, exception.NewInputException("page_size", "page size must be between 1 and 100"))
	}
	return inputErrors
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_wa.Message]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_wa.Message]](inputErrors)
	}
	if user.WA == nil || user.WA.PhoneNumber_ == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
	}
	if user.WA.PhoneNumber_.MetaPhoneNumberId != list.PhoneNumberId {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusUnauthorized, "you are not authorized to access this WhatsApp phone number")
	}
	messages, totalPages, totalCount, err := dependencies.UnitOfWork.WAMessageRepository().List(ctx, list.PhoneNumberId, list.CustomerWAId, list.CustomerMetaUserId, true, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	page := make([]dto_wa.Message, 0, len(messages))
	for _, message := range messages {
		page = append(page, dto_wa.NewMessage(message))
	}
	response := dto.NewPagedListResponse(page, totalPages, totalCount)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp messages",
		"Lists messages for a customer and WhatsApp business phone number.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/messages",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
