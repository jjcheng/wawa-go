package feature_ai_agent_skill

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	dao_ai_agent "github.com/jjcheng/wawa-go/internal/dao/ai_agent"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type AddCommon struct {
	ProfileId int32 `uri:"id" val:"required" description:"id of the agent profile"`
}

func (addCommon *AddCommon) Validate() []exception.InputException {
	var errors []exception.InputException
	if addCommon.ProfileId <= 0 {
		errors = append(errors, exception.NewInputException("profile_id", "invalid profile id"))
	}
	return errors
}

func (addCommon AddCommon) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if inputErrors := addCommon.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[any](inputErrors)
	}
	profile, err := dependencies.UnitOfWork.AIAgentProfileRepository().GetById(ctx, addCommon.ProfileId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "profile not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if profile.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	skills := []dao_ai_agent.Skill{
		{
			Title:       "How should customer address you.",
			Description: fmt.Sprintf("You are %s, the 24/7 AI assistant to help the customers to find out more about our business, book appointment etc...", profile.Name),
		},
		{
			Title:       "Welcome customers and identify what they need.",
			Description: "Greet the customer politely in their preferred language. Ask how you can help and clarify their request with one question at a time. Keep replies concise and avoid asking for unnecessary personal information.",
		},
		{
			Title:       "Answer questions about products, services, pricing, and availability.",
			Description: "Use only verified business information to answer product and service questions. Clarify the customer's needs before recommending an option. Never invent prices, availability, discounts, or guarantees. If information is unavailable, explain that and offer help from a human representative.",
		},
		{
			Title:       "Help customers with order status and delivery questions.",
			Description: "Ask for the minimum order reference needed to locate the customer's order. Share status and delivery information only from verified sources and after the required identity checks. Never invent tracking updates or promise a delivery date. If order information cannot be accessed, offer a human handoff.",
		},
		{
			Title:       "Handle complaints, return requests, and refund enquiries.",
			Description: "Acknowledge the customer's concern with empathy and ask for a concise description of the issue. Explain only the business's documented return and refund policies. Do not promise compensation or approve refunds without authorization. Escalate unresolved issues to a human representative with a summary of the concern.",
		},
		{
			Title:       "Escalate requests that require human assistance.",
			Description: "Offer human assistance when the customer requests it, when you lack reliable information, or when the issue requires authorization. Summarize the customer's request and relevant details without unnecessary sensitive data. Use an available handoff mechanism if provided; otherwise share verified support contact information. Never claim a transfer succeeded unless it is confirmed.",
		},
	}
	helper.Reverse(skills)
	// call create skill now
	for _, skill := range skills {
		request := Create{ProfileId: addCommon.ProfileId, Title: skill.Title, Description: skill.Description}
		response := request.Handle(ctx, user, dependencies)
		if !response.Success {
			return dto.Response[any]{ResponseBase: response.ResponseBase, Error: response.Error}
		}
	}
	return dto.NewSuccessResponse[any](nil)
}

func (AddCommon) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Add common AI agent skills",
		"Add common customer-service related skills. Stops on the first failure; skills already created are not rolled back. Repeated calls create additional skills.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/ai-agent/profiles/:id/common-skills",
		true,
		true,
		types.APITagBusinessAgent,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("profile not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
