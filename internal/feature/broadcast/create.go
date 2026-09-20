package feature_broadcast

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Create struct {
	Name         string              `json:"name"`
	SendDate     *time.Time          `json:"send_date"`
	WATemplateId string              `json:"wa_template_id"`
	SendTemplate dto_wa.SendTemplate `json:"send_template"`
	CustomerIds  []int32             `json:"customer_ids"`
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
		} else if (*create.SendDate).Before(time.Now().Add(1 * time.Minute)) {
			inputErrors = append(inputErrors, exception.NewInputException("send_date", "send date must be at least 1 minute from now"))
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

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_customer.Broadcast] {
	if user == nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusForbidden, "you are not authenticated")
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusUnauthorized, "you do not have a connected WhatsApp number")
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_customer.Broadcast](inputErrors)
	}
	exist, err := dependencies.UnitOfWork.BroadcastRepository().CheckNameExist(ctx, user.Id, create.Name)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if exist {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusBadRequest, "this broadcast name is already used")
	}
	customers, err := dependencies.UnitOfWork.CustomerRepository().ListByIds(ctx, user.Id, create.CustomerIds)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if len(customers) != len(create.CustomerIds) {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusBadRequest, "one or more customers were not found")
	}
	// check each customer, they all must have WAId
	for _, customer := range customers {
		if strings.TrimSpace(customer.WAId) == "" {
			return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusBadRequest, "one or more customers do not have a valid country code and phone number, please edit them")
		}
	}
	template, err := dependencies.Whatsapp.GetTemplate(ctx, create.WATemplateId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusBadRequest, "whatsapp template not found")
	}
	if template.Status != types.WATemplateStatusApproved {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusBadRequest, "whatsapp template is not approved")
	}
	// convert create.SendTemplate to map
	sendTemplatePayload := map[string]any{
		"components": create.SendTemplate.Components,
	}
	// for flow and quick_reply button, supply the broadcast token
	broadcastToken := fmt.Sprintf("broadcast_%s", strings.ReplaceAll(uuid.NewString(), "-", ""))
	// now generate a list of final payloads to send to meta
	var payloads []map[string]any
	for _, customer := range customers {
		// make deep copy of create.SendTemplate
		sendTemplate, err := helper.DeepCopy(create.SendTemplate)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, create)
			return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		payload, err := sendTemplate.FinalPayload(template, customer, broadcastToken)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, create)
			return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		payloads = append(payloads, payload)
	}
	// extract the attachment url if any
	var attachmentUrl string
	for _, component := range create.SendTemplate.Components {
		if component.Type != "header" {
			continue
		}
		for _, parameter := range component.Parameters {
			if parameter.Image != nil {
				attachmentUrl = parameter.Image.Link
				break
			} else if parameter.Video != nil {
				attachmentUrl = parameter.Video.Link
				break
			} else if parameter.Document != nil {
				attachmentUrl = parameter.Document.Link
				break
			}
		}
	}
	// start a transaction
	// create a broadcast first, get the broadcast id
	transaction := dependencies.UnitOfWork.BeginTransaction()
	committed := false
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	broadcast := dao_customer.Broadcast{
		Name:                create.Name,
		WATemplateId:        create.WATemplateId,
		UserId:              user.Id,
		Status:              types.BroadcastStatusPending,
		Token:               broadcastToken,
		SendTemplatePayload: sendTemplatePayload,
		AttachmentURL:       attachmentUrl,
		TemplatePayload:     template.Payload(),
		RecipientCount:      int32(len(customers)),
	}
	if create.SendDate != nil {
		broadcast.SendDate = *create.SendDate
	} else {
		// add 1 minute to current time to allow user to cancel
		broadcast.SendDate = time.Now().UTC().Add(1 * time.Minute)
	}
	if err := transaction.BroadcastRepository().Insert(ctx, &broadcast); err != nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	// store each payload as a BroadcastRecipient, use bulk insert
	recipients := make([]dao_customer.BroadcastRecipient, 0, len(customers))
	for index, customer := range customers {
		recipients = append(recipients, dao_customer.BroadcastRecipient{
			BroadcastId:  broadcast.Id,
			CustomerId:   customer.Id,
			Payload:      payloads[index],
			EncryptionID: uuid.NewString(),
		})
	}
	if err := transaction.BroadcastRecipientRepository().InsertBulk(ctx, recipients); err != nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[*dto_customer.Broadcast](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	committed = true
	// process after commit because this is like a process of event which happens in fc task
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		err := Process(ctx, broadcast.Id, 1, dependencies)
		if err != nil {
			dependencies.Logger.Warnf("failed to run broadcast worker locally: broadcast_id=%d err=%v", broadcast.Id, err)
		}
	}
	result := dto_customer.NewBroadcast(broadcast)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Start a broadcast",
		"Start a pending broadcast by the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/broadcasts",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("error seralizing template data", http.StatusInternalServerError)),
			feature.NewAPIError(*exception.NewCustomException("one or more customers were not found", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("one or more customers do not have a WhatsApp ID", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("this broadcast name is already used", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("whatsapp broadcast not found", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("whatsapp broadcast is not approved", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this business account", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
