package feature_wa_campaign

import (
	"context"
	"errors"
	"net/http"

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

type Get struct {
	Id int32 `uri:"id" val:"required" description:"id of the campaign"`
}

func (get *Get) Validate() []exception.InputException {
	if get.Id <= 0 {
		return []exception.InputException{exception.NewInputException("id", "invalid campaign id")}
	}
	return nil
}

func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Campaign] {
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.Campaign](http.StatusForbidden, "you are not authenticated")
	}
	if errors := get.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Campaign](errors)
	}
	campaign, err := dependencies.UnitOfWork.WACampaignRepository().GetById(ctx, get.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Campaign](http.StatusNotFound, "campaign not found")
		}
		return dto.NewFailedResponse[*dto_wa.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if campaign.UserId != user.Id {
		return dto.NewFailedResponse[*dto_wa.Campaign](http.StatusNotFound, "campaign not found")
	}
	customers, err := dependencies.UnitOfWork.CustomerRepository().ListByIds(ctx, user.Id, campaign.CustomerIds)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Campaign](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	customersById := make(map[int32]dto_customer.Customer, len(customers))
	for _, customer := range customers {
		customersById[customer.Id] = dto_customer.NewCustomer(customer)
	}
	orderedCustomers := make([]dto_customer.Customer, 0, len(customers))
	for _, customerId := range campaign.CustomerIds {
		if customer, ok := customersById[customerId]; ok {
			orderedCustomers = append(orderedCustomers, customer)
		}
	}
	result := dto_wa.NewCampaign(*campaign, orderedCustomers)
	return dto.NewSuccessResponse(&result)
}

func (Get) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp campaign",
		"Gets a campaign belonging to the authenticated user.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/campaigns/:id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("campaign not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
