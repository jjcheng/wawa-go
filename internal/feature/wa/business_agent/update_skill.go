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

type UpdateSkill struct {
	PhoneNumberId int32 `uri:"phone_number_id" val:"required" description:"id of the phone number"`
	service.AgentSkill
}

func (updateSkill *UpdateSkill) Validate() []exception.InputException {
	var inputErrors []exception.InputException
	updateSkill.ID = strings.TrimSpace(updateSkill.ID)
	if updateSkill.ID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("id", "missing ID"))
	}
	if updateSkill.PhoneNumberId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("phone_number_id", "invalid phone nmumber id"))
	}
	inputErrors = append(inputErrors, updateSkill.AgentSkill.Validate()...)
	return inputErrors
}

func (updateSkill UpdateSkill) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.AgentSkill] {
	if user == nil {
		return dto.NewFailedResponse[*service.AgentSkill](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.AgentSkill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := updateSkill.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.AgentSkill](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, updateSkill.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*service.AgentSkill](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*service.AgentSkill](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if user.Type == types.UserTypeOperator {
		if !helper.Any(user.WA.PhoneNumbers, func(assignedPhoneNumber dto_wa.PhoneNumber) bool {
			return assignedPhoneNumber.Id == updateSkill.PhoneNumberId
		}) {
			return dto.NewFailedResponse[*service.AgentSkill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
		}
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*service.AgentSkill](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	skill := &service.AgentSkill{
		ID:          updateSkill.ID,
		Title:       updateSkill.Title,
		Description: updateSkill.Description,
		Skill:       updateSkill.Skill,
		Channel:     "whatsapp",
	}
	updatedSkill, err := dependencies.Facebook.UpdateAgentSkill(ctx, phoneNumber.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken, skill)
	if err != nil {
		return dto.NewFailedResponse[*service.AgentSkill](http.StatusBadGateway, err.Error(), err)
	}
	return dto.NewSuccessResponse(updatedSkill)
}

func (UpdateSkill) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update WhatsApp business agent skill",
		"Updates a skill configured for a WhatsApp business agent.",
		types.HttpRequestTypeUriJSON,
		http.MethodPut,
		"/v1/wa/phone-numbers/:phone_number_id/business-agent/skills",
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
