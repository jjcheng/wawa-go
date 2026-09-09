package feature_campaign

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	Name         string                         `json:"name"`
	SendDate     *time.Time                     `json:"send_date"`
	WATemplateId string                         `json:"wa_template_id"`
	Components   []dto_wa.SendTemplateComponent `json:"components"`
	CustomerIds  []int32                        `json:"customer_ids"`
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
	exist, err := dependencies.UnitOfWork.CampaignRepository().CheckNameExist(ctx, user.Id, create.Name)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if exist {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusBadRequest, "this campaign name is already used")
	}
	customers, err := dependencies.UnitOfWork.CustomerRepository().ListByIds(ctx, user.Id, create.CustomerIds)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if len(customers) != len(create.CustomerIds) {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusBadRequest, "one or more customers were not found")
	}
	businessPortfolio, _, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusUnauthorized, "you are not authorized to access this business account")
		}
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	template, err := dependencies.Whatsapp.GetTemplate(ctx, create.WATemplateId, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusBadRequest, "whatsapp template not found")
	}
	if template.Status != types.WATemplateStatusApproved {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusBadRequest, "whatsapp template is not approved")
	}
	campaign := dao_customer.Campaign{
		Name:         create.Name,
		WATemplateId: create.WATemplateId,
		CustomerIds:  dao_customer.CustomerIDs(create.CustomerIds),
		UserId:       user.Id,
		Status:       types.CampaignStatusPending,
		Payload: map[string]any{
			"template": map[string]any{
				"name": template.Name,
				"language": map[string]string{
					"code": template.Language,
				},
				"components": create.Components,
			},
		},
	}
	if create.SendDate != nil {
		campaign.SendDate = sql.NullTime{Time: *create.SendDate, Valid: true}
	}
	if err := dependencies.UnitOfWork.CampaignRepository().Insert(ctx, &campaign); err != nil {
		return dto.NewFailedResponse[*dto_customer.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	result := dto_customer.NewCampaign(campaign, nil)
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Start a campaign",
		"Start a pending campaign by the authenticated user.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/campaigns",
		true,
		true,
		types.APITagCustomer,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("one or more customers were not found", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("this campaign name is already used", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("whatsapp template not found", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("whatsapp template is not approved", http.StatusBadRequest)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this business account", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
