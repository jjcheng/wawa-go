package feature_wa_business_agent

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
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

type CreateKeyword struct {
	PhoneNumberId     int32  `uri:"phone_number_id" val:"required" description:"if of the phone number"`
	Keyword           string `json:"keyword" val:"required" description:"keyword to monitor"`
	NotificationTitle string `json:"notification_title" val:"required" description:"title of the notification to send if a keyword is detected"`
}

func (createKeyword *CreateKeyword) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	if createKeyword.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone number id"))
	}
	createKeyword.Keyword = strings.TrimSpace(createKeyword.Keyword)
	if createKeyword.Keyword == "" {
		inputErrors = append(inputErrors, exception.NewInputException("keyword", "missing keyword"))
	} else if len(createKeyword.Keyword) < 10 {
		inputErrors = append(inputErrors, exception.NewInputException("keyword", "keyword must be at least 10 characters"))
	}
	createKeyword.NotificationTitle = strings.TrimSpace(createKeyword.NotificationTitle)
	if createKeyword.NotificationTitle == "" {
		inputErrors = append(inputErrors, exception.NewInputException("notification_title", "missing notification title"))
	}
	return inputErrors
}

func (createKeyword CreateKeyword) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessAgentKeyword] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := createKeyword.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessAgentKeyword](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, createKeyword.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == createKeyword.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	keyword := dao_wa.BusinessAgentKeyword{
		PhoneNumberId:     createKeyword.PhoneNumberId,
		Keyword:           createKeyword.Keyword,
		NotificationTitle: createKeyword.NotificationTitle,
	}
	if err := dependencies.UnitOfWork.WABusinessAgentKeywordRepository().Insert(ctx, &keyword); err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_wa.NewBusinessAgentKeyword(keyword)
	return dto.NewSuccessResponse(&result)
}

func (CreateKeyword) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp business agent keyword",
		"Adds a keyword to monitor for a WhatsApp business agent.",
		types.HttpRequestTypeUriJSON,
		http.MethodPost,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/keywords",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
