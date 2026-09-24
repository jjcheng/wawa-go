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
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_wa.Message]](inputErrors)
	}
	customer, err := dependencies.UnitOfWork.CustomerRepository().GetById(ctx, list.CustomerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusNotFound, "customer not found", nil)
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == customer.PhoneNumberId
	}) {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	messages, totalPages, totalCount, err := dependencies.UnitOfWork.WAMessageRepository().List(ctx, customer.PhoneNumberId, customer.Id, true, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	messageIds := helper.Map(messages, func(message dao_wa.Message) int32 { return message.Id })
	// get the broadcasts
	broadcasts, err := dependencies.UnitOfWork.BroadcastRepository().ListByMessageIds(ctx, messageIds)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_wa.Message]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
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
			var broadcast *dao_customer.Broadcast
			if len(broadcasts) == 1 {
				broadcast = &broadcasts[0]
			} else if sendTemplateMap, ok := sendTemplatePayload.(map[string]any); ok {
				if templateName, ok := sendTemplateMap["name"].(string); ok && templateName != "" {
					broadcast = helper.First(broadcasts, func(c dao_customer.Broadcast) bool {
						name, _ := c.TemplatePayload["name"].(string)
						return name == templateName
					})
				}
			}
			if broadcast == nil && len(broadcasts) > 0 {
				broadcast = &broadcasts[0]
			}
			if broadcast == nil {
				continue
			}
			template, err := helper.ConvertJSON[dto_wa.Template](broadcast.TemplatePayload)
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
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
