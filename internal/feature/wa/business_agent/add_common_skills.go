package feature_wa_business_agent

import (
	"context"
	"errors"
	"fmt"
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

type AddCommonSkills struct {
	PhoneNumberId int32 `uri:"phone_number_id" json:"phone_number_id" val:"required" description:"id of the phone number"`
}

func (addCommonSkills *AddCommonSkills) Validate() []exception.InputException {
	var errors []exception.InputException
	if addCommonSkills.PhoneNumberId <= 0 {
		errors = append(errors, exception.NewInputException("phone_number_id", "missing phone number id"))
	}
	return errors
}

func (addCommonSkills AddCommonSkills) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := addCommonSkills.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, addCommonSkills.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// for operator need to check if the number is managed, otherwise check business account
	if user.Type != types.UserTypeMaster && !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == addCommonSkills.PhoneNumberId
	}) {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	} else if phoneNumber.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	skills := []service.AgentSkill{
		{
			Title:       "your-persona",
			Description: "How should customer address you.",
			Skill:       fmt.Sprintf("You are %s, the 24/7 AI assistant to help the customers to find out more about our buisness, book appointment.", phoneNumber.Name),
		},
		{
			Title:       "customer-greeting",
			Description: "Welcome customers and identify what they need.",
			Skill:       "Greet the customer politely in their preferred language. Ask how you can help and clarify their request with one question at a time. Keep replies concise and avoid asking for unnecessary personal information.",
		},
		{
			Title:       "product-and-service-enquiries",
			Description: "Answer questions about products, services, pricing, and availability.",
			Skill:       "Use only verified business information to answer product and service questions. Clarify the customer's needs before recommending an option. Never invent prices, availability, discounts, or guarantees. If information is unavailable, explain that and offer help from a human representative.",
		},
		{
			Title:       "order-and-delivery-enquiries",
			Description: "Help customers with order status and delivery questions.",
			Skill:       "Ask for the minimum order reference needed to locate the customer's order. Share status and delivery information only from verified sources and after the required identity checks. Never invent tracking updates or promise a delivery date. If order information cannot be accessed, offer a human handoff.",
		},
		{
			Title:       "complaints-and-returns",
			Description: "Handle complaints, return requests, and refund enquiries.",
			Skill:       "Acknowledge the customer's concern with empathy and ask for a concise description of the issue. Explain only the business's documented return and refund policies. Do not promise compensation or approve refunds without authorization. Escalate unresolved issues to a human representative with a summary of the concern.",
		},
		{
			Title:       "human-support-handoff",
			Description: "Escalate requests that require human assistance.",
			Skill:       "Offer human assistance when the customer requests it, when you lack reliable information, or when the issue requires authorization. Summarize the customer's request and relevant details without unnecessary sensitive data. Use an available handoff mechanism if provided; otherwise share verified support contact information. Never claim a transfer succeeded unless it is confirmed.",
		},
	}
	helper.Reverse(skills)
	// call create skill now
	for _, skill := range skills {
		request := CreateSkill{PhoneNumberId: addCommonSkills.PhoneNumberId, AgentSkill: skill}
		response := request.Handle(ctx, user, dependencies)
		if !response.Success {
			return dto.Response[any]{ResponseBase: response.ResponseBase, Error: response.Error}
		}
	}
	return dto.NewSuccessResponse[any](nil)
}

func (AddCommonSkills) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Add common WhatsApp business agent skills",
		"Creates five common customer-service skills. Stops on the first failure; skills already created are not rolled back. Repeated calls create additional skills.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/wa/business-agent/phone-numbers/:phone_number_id/common-skills",
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
