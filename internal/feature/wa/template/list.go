package feature_wa_template

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type List struct {
	MetaWABAId string `form:"meta_waba_id" description:"optional meta WABA id"`
}

func (list *List) Validate() []exception.InputException {
	list.MetaWABAId = strings.TrimSpace(list.MetaWABAId)
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.Template] {
	if errors := list.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.Template](errors)
	}
	phoneNumbers, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[[]dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if len(phoneNumbers) == 0 {
		return dto.NewFailedResponse[[]dto_wa.Template](http.StatusNotFound, "no phone number for this user")
	}
	seen := make(map[string]struct{}, len(phoneNumbers))
	assignedWABAIDs := make([]string, 0, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.MetaWABAId == "" {
			continue
		}
		if _, exists := seen[phoneNumber.MetaWABAId]; exists {
			continue
		}
		seen[phoneNumber.MetaWABAId] = struct{}{}
		assignedWABAIDs = append(assignedWABAIDs, phoneNumber.MetaWABAId)
	}
	wabaIDs := assignedWABAIDs
	if list.MetaWABAId != "" {
		if _, exists := seen[list.MetaWABAId]; !exists {
			return dto.NewFailedResponse[[]dto_wa.Template](http.StatusUnauthorized, "you are not authorized to access this WABA")
		}
		wabaIDs = []string{list.MetaWABAId}
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, phoneNumbers[0].MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.Template](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[[]dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	templates := make([]dto_wa.Template, 0)
	for _, wabaID := range wabaIDs {
		wabaTemplates, err := dependencies.Whatsapp.ListTemplates(ctx, wabaID, businessPortfolio.AccessToken)
		if err != nil {
			return dto.NewFailedResponse[[]dto_wa.Template](http.StatusBadGateway, types.ExceptionMessageBadGateway)
		}
		for index := range wabaTemplates {
			wabaTemplates[index].MetaWABAId = wabaID
		}
		templates = append(templates, wabaTemplates...)
	}
	return dto.NewSuccessResponse(templates)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp templates",
		"Lists WhatsApp message templates available to the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/wa/v1/templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
			feature.NewAPIError(*exception.NewCustomException("no phone number for this user", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
		},
	)
}
