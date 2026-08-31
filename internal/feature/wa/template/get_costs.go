package feature_wa_template

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type GetCosts struct {
	MetaWABAId  string   `form:"meta_waba_id" val:"required" description:"Meta WABA id"`
	Start       string   `form:"start" val:"required" description:"Analytics start date in YYYY-MM-DD format"`
	End         string   `form:"end" val:"required" description:"Analytics end date in YYYY-MM-DD format"`
	TemplateIds []string `form:"template_ids" val:"required" description:"one to ten numeric template IDs"`
}

func (getCosts *GetCosts) Validate() []exception.InputException {
	getCosts.MetaWABAId = strings.TrimSpace(getCosts.MetaWABAId)
	getCosts.Start = strings.TrimSpace(getCosts.Start)
	getCosts.End = strings.TrimSpace(getCosts.End)
	inputErrors := []exception.InputException{}
	if getCosts.MetaWABAId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("meta_waba_id", "missing Meta WABA id"))
	}
	startDate, startErr := time.Parse("2006-01-02", getCosts.Start)
	endDate, endErr := time.Parse("2006-01-02", getCosts.End)
	if startErr != nil {
		inputErrors = append(inputErrors, exception.NewInputException("start", "start must use YYYY-MM-DD format"))
	}
	if endErr != nil {
		inputErrors = append(inputErrors, exception.NewInputException("end", "end must use YYYY-MM-DD format"))
	}
	if startErr == nil && endErr == nil && !endDate.After(startDate) {
		inputErrors = append(inputErrors, exception.NewInputException("end", "end must be later than start"))
	}
	if startErr == nil && startDate.Before(time.Now().UTC().AddDate(0, 0, -90)) {
		inputErrors = append(inputErrors, exception.NewInputException("start", "start must be within the last 90 days"))
	}
	if len(getCosts.TemplateIds) == 0 || len(getCosts.TemplateIds) > 10 {
		inputErrors = append(inputErrors, exception.NewInputException("template_ids", "template_ids must contain between 1 and 10 IDs"))
	}
	for _, templateId := range getCosts.TemplateIds {
		if _, err := strconv.ParseUint(strings.TrimSpace(templateId), 10, 64); err != nil {
			inputErrors = append(inputErrors, exception.NewInputException("template_ids", "template IDs must be numeric"))
			break
		}
	}
	return inputErrors
}

func (getCosts GetCosts) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.TemplateAnalytics] {
	if validationErrors := getCosts.Validate(); len(validationErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.TemplateAnalytics](validationErrors)
	}
	businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetValidBusinessAccount(ctx, user.Id, getCosts.MetaWABAId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusUnauthorized, "you are not authorized to access this WABA")
		}
		return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	businessPortfolio, err := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, businessAccount.MetaBusinessPortfolioId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusNotFound, "business portfolio not found")
		}
		return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	analytics, err := dependencies.Whatsapp.GetTemplateCosts(ctx, getCosts.MetaWABAId, getCosts.Start, getCosts.End, getCosts.TemplateIds, businessPortfolio.AccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, getCosts.MetaWABAId, getCosts.TemplateIds)
		return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	return dto.NewSuccessResponse(analytics)
}

func (GetCosts) APISettings() feature.APISettings {
	return feature.NewAPISettings("Get WhatsApp template costs", "Gets cost analytics for WhatsApp message templates.", types.HttpRequestTypeQuery, http.MethodGet, "/v1/wa/templates/costs", true, true, types.APITagWA, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
