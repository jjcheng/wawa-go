package feature_campaign

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Create struct {
	Name         string                                 `json:"name"`
	SendDate     *time.Time                             `json:"send_date"`
	WATemplateId string                                 `json:"wa_template_id"`
	Components   []feature_wa_message.TemplateComponent `json:"components"`
	CustomerIds  []int32                                `json:"customer_ids"`
}

func (create *Create) Validate() []exception.InputException {
	create.Name = strings.TrimSpace(create.Name)
	create.WATemplateId = strings.TrimSpace(create.WATemplateId)
	inputErrors := []exception.InputException{}
	if create.Name == "" {
		inputErrors = append(inputErrors, exception.NewInputException("name", "missing name"))
	}
	if create.SendDate != nil {
		if (*create.SendDate).IsZero() {
			inputErrors = append(inputErrors, exception.NewInputException("send_date", "missing send date"))
		} else if (*create.SendDate).Before(time.Now().Add(5 * time.Minute)) {
			inputErrors = append(inputErrors, exception.NewInputException("send_date", "send date must be at least 5 minutes from now"))
		}
	}
	if create.WATemplateId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("wa_template_id", "missing whatsapp template id"))
	}
	if len(create.CustomerIds) == 0 {
		inputErrors = append(inputErrors, exception.NewInputException("customer_ids", "at least one customer is required"))
	}
	seenCustomerIds := make(map[int32]struct{}, len(create.CustomerIds))
	for _, customerId := range create.CustomerIds {
		if customerId <= 0 {
			inputErrors = append(inputErrors, exception.NewInputException("customer_ids", "customer ids must be positive"))
			break
		}
		if _, ok := seenCustomerIds[customerId]; ok {
			inputErrors = append(inputErrors, exception.NewInputException("customer_ids", "customer ids must be unique"))
			break
		}
		seenCustomerIds[customerId] = struct{}{}
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Campaign] {
	if user == nil {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Campaign](inputErrors)
	}
	customers, err := dependencies.UnitOfWork.CustomerRepository().ListByIds(ctx, user.Id, create.CustomerIds)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if len(customers) != len(create.CustomerIds) {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusBadRequest, "one or more customers were not found")
	}
	// campaign := dao_wa.Campaign{
	// 	MessageBase: dao_wa.MessageBase{
	// 		AttachmentUrl:     create.AttachmentUrl,
	// 		AttachmentType:    create.AttachmentType,
	// 		LocationName:      create.LocationName,
	// 		LocationAddress:   create.LocationAddress,
	// 		LocationLatitude:  create.LocationLatitude,
	// 		LocationLongitude: create.LocationLongitude,
	// 		HeaderText:        create.HeaderText,
	// 		BodyText:          create.BodyText,
	// 		FooterText:        create.FooterText,
	// 	},
	// 	Name:         create.Name,
	// 	WATemplateId: create.WATemplateId,
	// 	CustomerIds:  create.CustomerIds,
	// 	UserId:       user.Id,
	// 	Status:       types.WACampaignStatusPending,
	// 	Archived:     false,
	// }
	// if create.SendDate != nil {
	// 	campaign.SendDate = sql.NullTime{
	// 		Time:  *create.SendDate,
	// 		Valid: true,
	// 	}
	// }
	// if err := dependencies.UnitOfWork.WACampaignRepository().Insert(ctx, &campaign); err != nil {
	// 	return dto.NewFailedResponse[*dto_wa.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	// }
	// result := dto_wa.NewCampaign(campaign, nil)
	// return dto.NewSuccessResponse(&result)
	return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusNotImplemented, "")
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Start WhatsApp campaign",
		"Start a pending campaign by the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/campaigns",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("one or more customers were not found", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
