package feature_wa_template

import (
	"context"
	"errors"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	dto_wa.TemplateBase
}

func (create *Create) Validate() []exception.InputException {
	return create.TemplateBase.Validate()
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Template] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Template](errors)
	}
	businessPortfolio, businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusUnauthorized, "you are not authorized to access this WABA")
		}
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	metaWABAId := businessAccount.MetaWABAId
	template, err := dependencies.Whatsapp.CreateTemplate(ctx, metaWABAId, create.Payload(), businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewSuccessResponse(template)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp template",
		"Creates a WhatsApp message template for an authorized WABA.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
