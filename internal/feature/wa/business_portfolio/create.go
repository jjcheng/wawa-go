package feature_wa_business_portfolio

import (
	"context"
	"errors"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Create struct {
	MetaBusinessPortfolioId string `json:"meta_business_portfolio_id" val:"required" description:"return in embedded signup"`
	TemporaryCode           string `json:"temporary_code" val:"required" description:"temporary code returned in embedded signup"`
}

func (create *Create) Validate() []exception.InputException {
	var errors []exception.InputException
	create.MetaBusinessPortfolioId = strings.TrimSpace(create.MetaBusinessPortfolioId)
	create.TemporaryCode = strings.TrimSpace(create.TemporaryCode)
	if create.MetaBusinessPortfolioId == "" {
		errors = append(errors, exception.NewInputException("meta_business_portfolio_id", "missing Meta business portfolio id"))
	}
	if create.TemporaryCode == "" {
		errors = append(errors, exception.NewInputException("temporary_token", "missing temporary code"))
	}
	return errors
}

func (create Create) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.BusinessPortfolio] {
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.BusinessPortfolio](errors)
	}
	existing, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, create.MetaBusinessPortfolioId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessTokenResponse, err := dependencies.Whatsapp.GetBusinessAccessToken(ctx, create.TemporaryCode)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	businessName, err := dependencies.Whatsapp.GetBusinessName(ctx, create.MetaBusinessPortfolioId, businessTokenResponse.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	if existing != nil {
		if existing.Name != businessName {
			existing.Name = businessName
			existing.AccessToken = businessTokenResponse.AccessToken
			existing.AccessTokenExpiresIn = int32(businessTokenResponse.ExpiresIn)
			if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Update(ctx, existing); err != nil {
				return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
			}
		}
		d := dto_wa.NewBusinessPortfolio(*existing, false)
		return dto.NewSuccessResponse(&d)
	}
	businessPortfolio := dao_wa.BusinessPortfolio{
		MetaBusinessPortfolioId: create.MetaBusinessPortfolioId,
		Name:                    businessName,
		AccessToken:             businessTokenResponse.AccessToken,
		AccessTokenExpiresIn:    int32(businessTokenResponse.ExpiresIn),
	}
	if err := dependencies.UnitOfWork.WABusinessPortfolioRepository().Insert(ctx, &businessPortfolio); err != nil {
		return dto.NewFailedResponse[*dto_wa.BusinessPortfolio](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	d := dto_wa.NewBusinessPortfolio(businessPortfolio, true)
	return dto.NewSuccessResponse(&d)
}
