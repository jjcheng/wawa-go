package feature_wa_business_agent

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
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type UpdateKeyword struct {
	PhoneNumberId     int32  `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	Id                int32  `uri:"id" val:"required" description:"id of the keyword"`
	Keyword           string `json:"keyword" val:"required" description:"keyword to monitor"`
	NotificationTitle string `json:"notification_title" val:"required" description:"title of the notification to send when a keyword is detected"`
}

func (updateKeyword *UpdateKeyword) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if updateKeyword.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	if updateKeyword.Id <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("id", "invalid keyword id"))
	}
	updateKeyword.Keyword = strings.TrimSpace(updateKeyword.Keyword)
	if updateKeyword.Keyword == "" {
		inputErrors = append(inputErrors, exception.NewInputException("keyword", "missing keyword"))
	}
	return inputErrors
}

func (updateKeyword UpdateKeyword) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAgentKeyword] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := updateKeyword.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAgentKeyword](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, updateKeyword.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == updateKeyword.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	keyword, err := dependencies.UnitOfWork.WABusinessAgentKeywordRepository().GetById(ctx, updateKeyword.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusNotFound, "keyword not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if keyword.PhoneNumberId != updateKeyword.PhoneNumberId {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	keyword.Keyword = updateKeyword.Keyword
	keyword.NotificationTitle = updateKeyword.NotificationTitle
	if err := dependencies.UnitOfWork.WABusinessAgentKeywordRepository().Update(ctx, keyword); err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_wa.NewBusinessAgentKeyword(*keyword)
	return dto.NewSuccessResponse(&result)
}

func (UpdateKeyword) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business agent keyword",
		"Updates a keyword monitored by a WhatsApp business agent.",
		types.HttpRequestTypeUriJSON,
		http.MethodPut,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/keywords/:id",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("keyword not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
