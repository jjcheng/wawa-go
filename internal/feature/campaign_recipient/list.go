package feature_campaign_recipient

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_customer "github.com/jjcheng/wawa-go/internal/dto/customer"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type List struct {
	Name       string                `form:"name" description:"name of the customer to filter"`
	CampaignId int32                 `form:"campaign_id" val:"required" description:"id of the campaign"`
	Status     types.WAMessageStatus `form:"status" description:"optional recipient status filter, use message status each recipient is tied to a message"`
	Page       int                   `form:"page" description:"page number from 1"`
	PageSize   int                   `form:"page_size" description:"number per page"`
}

func (list *List) Validate() []exception.InputException {
	list.Name = strings.TrimSpace(list.Name)
	if list.CampaignId <= 0 {
		return []exception.InputException{exception.NewInputException("campaign_id", "invalid campaign id")}
	}
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 25
	}
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto.ListResponse[dto_customer.CampaignRecipient]] {
	if user == nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.CampaignRecipient]](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := list.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto.ListResponse[dto_customer.CampaignRecipient]](inputErrors)
	}
	campaign, err := dependencies.UnitOfWork.CampaignRepository().GetById(ctx, list.CampaignId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto.ListResponse[dto_customer.CampaignRecipient]](http.StatusNotFound, "campaign not found")
		}
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.CampaignRecipient]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if campaign.UserId != user.Id {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.CampaignRecipient]](http.StatusNotFound, "campaign not found")
	}
	recipients, totalCount, totalPages, err := dependencies.UnitOfWork.CampaignRecipientRepository().ListByCampaignId(ctx, list.CampaignId, list.Name, list.Status, false, list.Page, list.PageSize)
	if err != nil {
		return dto.NewFailedResponse[*dto.ListResponse[dto_customer.CampaignRecipient]](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	items := make([]dto_customer.CampaignRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		items = append(items, dto_customer.NewCampaignRecipient(recipient))
	}
	response := dto.NewPagedListResponse(items, totalPages, totalCount)
	return dto.NewSuccessResponse(&response)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List campaign recipients",
		"Lists recipients belonging to an authenticated user's campaign.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/campaigns/recipients",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
