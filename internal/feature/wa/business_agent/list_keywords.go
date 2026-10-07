package feature_wa_business_agent

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

type ListKeywords struct {
	PhoneNumberId int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (listKeywords *ListKeywords) Validate() []exception.InputException {
	if listKeywords.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (listKeywords ListKeywords) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.BusinessAgentKeyword] {
	if user == nil {
		return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := listKeywords.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.BusinessAgentKeyword](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, listKeywords.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == listKeywords.PhoneNumberId
		}) {
			return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	keywords, err := dependencies.UnitOfWork.WABusinessAgentKeywordRepository().ListByPhoneNumberId(ctx, listKeywords.PhoneNumberId)
	if err != nil {
		return dto.NewFailedResponse[[]dto_wa.BusinessAgentKeyword](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := make([]dto_wa.BusinessAgentKeyword, len(keywords))
	for index, keyword := range keywords {
		result[index] = dto_wa.NewBusinessAgentKeyword(keyword)
	}
	return dto.NewSuccessResponse(result)
}

func (ListKeywords) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp business agent keywords",
		"Lists the keywords configured for a WhatsApp business agent.",
		types.HttpRequestTypeUri,
		http.MethodGet,
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
