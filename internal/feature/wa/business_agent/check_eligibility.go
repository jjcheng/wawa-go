package feature_wa_business_agent

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type CheckEligibility struct {
	PhoneNumberId int32 `form:"phone_number_id" val:"required" description:"id of the phone number"`
}

type CheckEligibilityResult struct {
	IsEligible bool `json:"is_eligible"`
}

func (checkEligibility *CheckEligibility) Validate() []exception.InputException {
	if checkEligibility.PhoneNumberId <= 0 {
		return []exception.InputException{exception.NewInputException("phone_number_id", "invalid phone number id")}
	}
	return nil
}

func (checkEligibility CheckEligibility) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*CheckEligibilityResult] {
	if user == nil {
		return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolioAccessToken == "" {
		return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := checkEligibility.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*CheckEligibilityResult](inputErrors)
	}
	// only master can setup business agent
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusUnauthorized, types.ExceptionMessageInternalServerError, nil)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, checkEligibility.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	isEligible, err := dependencies.Facebook.CheckAgentEligibility(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*CheckEligibilityResult](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(&CheckEligibilityResult{IsEligible: isEligible})
}

func (CheckEligibility) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Check WhatsApp business agent eligibility",
		"Checks whether a WhatsApp phone number is eligible for the business agent.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/business-agent/eligibility",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("phone number not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
