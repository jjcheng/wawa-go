package feature_wa_user_phone_number

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

type List struct {
}

func (list *List) Validate() []exception.InputException {
	return nil
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.PhoneNumber] {
	if errors := list.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.PhoneNumber](errors)
	}
	phoneNumbers, ex := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if ex != nil {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	// populate the PhoneNumber's BusinessPortfolio and BusinessAccount object using the newly created func
	// first gather distinct MetaBusinessPortfolioId and MetaWABAId first
	portfolioIDs := make([]string, 0, len(phoneNumbers))
	wabaIDs := make([]string, 0, len(phoneNumbers))
	seenPortfolioIDs := make(map[string]struct{})
	seenWABAIDs := make(map[string]struct{})
	for _, phoneNumber := range phoneNumbers {
		if _, ok := seenPortfolioIDs[phoneNumber.MetaBusinessPortfolioId]; !ok && phoneNumber.MetaBusinessPortfolioId != "" {
			seenPortfolioIDs[phoneNumber.MetaBusinessPortfolioId] = struct{}{}
			portfolioIDs = append(portfolioIDs, phoneNumber.MetaBusinessPortfolioId)
		}
		if _, ok := seenWABAIDs[phoneNumber.MetaWABAId]; !ok && phoneNumber.MetaWABAId != "" {
			seenWABAIDs[phoneNumber.MetaWABAId] = struct{}{}
			wabaIDs = append(wabaIDs, phoneNumber.MetaWABAId)
		}
	}
	portfolios, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().ListByMetaBusinessPortfolioIds(ctx, portfolioIDs)
	if ex != nil {
		return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	accounts, ex := dependencies.UnitOfWork.WABusinessAccountRepository().ListByMetaWABAIds(ctx, wabaIDs)
	if ex != nil {
		if errors.Is(ex, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusNotFound, "business account not found")
		}
		return dto.NewFailedResponse[[]dto_wa.PhoneNumber](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	portfolioByID := make(map[string]dto_wa.BusinessPortfolio, len(portfolios))
	for _, portfolio := range portfolios {
		portfolioByID[portfolio.MetaBusinessPortfolioId] = dto_wa.NewBusinessPortfolio(portfolio, false)
	}
	accountByID := make(map[string]dto_wa.BusinessAccount, len(accounts))
	for _, account := range accounts {
		accountByID[account.MetaWABAId] = dto_wa.NewBusinessAccount(account)
	}
	result := make([]dto_wa.PhoneNumber, 0, len(phoneNumbers))
	for _, phoneNumber := range phoneNumbers {
		phoneNumberDTO := dto_wa.NewPhoneNumber(phoneNumber)
		if portfolio, ok := portfolioByID[phoneNumber.MetaBusinessPortfolioId]; ok {
			phoneNumberDTO.BusinessPortfolio = &portfolio
		}
		if account, ok := accountByID[phoneNumber.MetaWABAId]; ok {
			phoneNumberDTO.BusinessAccount = &account
		}
		result = append(result, phoneNumberDTO)
	}
	return dto.NewSuccessResponse(result)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List user WhatsApp phone numbers",
		"Lists the WhatsApp phone numbers assigned to the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/wa/v1/user-phone-numbers",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
			feature.NewAPIError(*exception.NewCustomException("business account not found", http.StatusNotFound)),
		},
	)
}
