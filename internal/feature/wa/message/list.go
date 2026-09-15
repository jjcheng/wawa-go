package feature_wa_message

import (
	"context"
	"errors"
	"net/http"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
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

type List struct {
	CustomerId int32 `form:"customer_id" val:"required" description:"id of the customer"`
	Page       int   `form:"page" description:"page number from 1"`
	PageSize   int   `form:"page_size" description:"page size, default 50"`
}

func (list *List) Validate() []exception.InputException {
	if list.Page == 0 {
		list.Page = 1
	}
	if list.PageSize == 0 {
		list.PageSize = 50
	}
	inputErrors := []exception.InputException{}
	if list.CustomerId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("customer_id", "missing customer ID"))
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
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusUnauthorized, "you are not authorized")
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, list.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusNotFound, "customer not found")
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if customer.UserId != user.Id {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusNotFound, "customer not found")
	}
	messages, totalPages, totalCount, err := dependencies.UnitOfWork.WAMessageRepository().List(ctx, user.WA.PhoneNumber_.Id, customer.Id, true, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	messageIds := helper.Map(messages, func(message dao_wa.Message) int32 { return message.Id })
	// get the campaigns
	campaigns, err := dependencies.UnitOfWork.CampaignRepository().ListByMessageIds(ctx, messageIds)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	// bind the html for the messages
	page := make([]dto_wa.Message, 0, len(messages))
	for _, message := range messages {
		result := dto_wa.NewMessage(message)
		if message.Type == "template" {
			sendTemplatePayload, ok := message.Payload["template"]
			if !ok {
				continue
			}
			sendTemplate, err := helper.ConvertJSON[dto_wa.SendTemplate](sendTemplatePayload)
			if err != nil {
				continue
			}
			var campaign *dao_customer.Campaign
			if len(campaigns) == 1 {
				campaign = &campaigns[0]
			} else if sendTemplateMap, ok := sendTemplatePayload.(map[string]any); ok {
				if templateName, ok := sendTemplateMap["name"].(string); ok && templateName != "" {
					campaign = helper.First(campaigns, func(c dao_customer.Campaign) bool {
						name, _ := c.TemplatePayload["name"].(string)
						return name == templateName
					})
				}
			}
			if campaign == nil && len(campaigns) > 0 {
				campaign = &campaigns[0]
			}
			if campaign == nil {
				continue
			}
			template, err := helper.ConvertJSON[dto_wa.Template](campaign.TemplatePayload)
			if err != nil {
				continue
			}
			template.ApplySendTemplate(sendTemplate)
			result.PreviewHTML = template.HTML(true, false)
			result.PreviewDarkHTML = template.HTML(true, true)
		}
		page = append(page, result)
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
