package feature_public

import (
	"context"
	"net/http"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

// do a rotation of user to be featured as the whatsapp contact
type GetWebsiteWALink struct {
}

func (getWebsiteWALink *GetWebsiteWALink) Validate() []exception.InputException {
	return nil
}

func (getWebsiteWALink GetWebsiteWALink) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
	if errors := getWebsiteWALink.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[string](http.StatusNotFound, "website not found")
	}
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetById(ctx, website.BusinessAccountId)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, "business account not found")
	}
	// TODO: rotate the user, for now just pick the first one
	allUsers, err := dependencies.UnitOfWork.AccountUserRepository().ListByBusinessAccountId(ctx, businessAccount.Id, nil, true)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, "failed to get users")
	}
	activeUsers := helper.Filter(allUsers, func(u dao_account.User) bool {
		return u.Status == types.UserStatusActive
	})
	if len(activeUsers) == 0 {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, "failed to get user")
	}
	// get phone number
	phoneNumber, _, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByUserId(ctx, activeUsers[0].Id)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, "failed to get phone number")
	}
	phoneNumberDTO := dto_wa.NewPhoneNumber(*phoneNumber)
	waLink := phoneNumberDTO.WALink()
	return dto.NewSuccessResponse(waLink)
}

func (GetWebsiteWALink) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp link of the website",
		"Use rotation strategy to get the next user's WhatsApp link for the website.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/public/wa-link",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
