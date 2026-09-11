package feature_wa_template

import (
	"context"
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
)

type GetUsage struct {
	Start       string                       `form:"start" val:"required" description:"Analytics start date in YYYY-MM-DD format"`
	End         string                       `form:"end" val:"required" description:"Analytics end date in YYYY-MM-DD format"`
	TemplateIds []string                     `form:"template_ids" val:"required" description:"one to ten numeric template IDs"`
	Granularity types.WAAnalyticsGranularity `form:"granularity" val:"required" description:"analytics granularity: HALF_HOUR, DAY, or MONTH"`
}

func (getUsage *GetUsage) Validate() []exception.InputException {
	getUsage.Start = strings.TrimSpace(getUsage.Start)
	getUsage.End = strings.TrimSpace(getUsage.End)
	inputErrors := []exception.InputException{}
	startDate, startErr := time.Parse("2006-01-02", getUsage.Start)
	endDate, endErr := time.Parse("2006-01-02", getUsage.End)
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
	if len(getUsage.TemplateIds) == 0 || len(getUsage.TemplateIds) > 10 {
		inputErrors = append(inputErrors, exception.NewInputException("template_ids", "template_ids must contain between 1 and 10 IDs"))
	}
	for _, templateId := range getUsage.TemplateIds {
		if _, err := strconv.ParseUint(strings.TrimSpace(templateId), 10, 64); err != nil {
			inputErrors = append(inputErrors, exception.NewInputException("template_ids", "template IDs must be numeric"))
			break
		}
	}
	if getUsage.Granularity == "DAY" {
		// somehow template analytics uses different
		getUsage.Granularity = "DAILY"
	}
	return inputErrors
}

func (getUsage GetUsage) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.TemplateAnalytics] {
	if validationErrors := getUsage.Validate(); len(validationErrors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.TemplateAnalytics](validationErrors)
	}
	if user.WA == nil || user.WA.BusinessAccount == nil {
		return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusUnauthorized, "you are not authorized to access this WABA")
	}
	metaWABAId := user.WA.BusinessAccount.MetaWABAId
	analytics, err := dependencies.Whatsapp.GetTemplateUsage(ctx, metaWABAId, getUsage.Start, getUsage.End, getUsage.TemplateIds, getUsage.Granularity, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[[]dto_wa.TemplateAnalytics](http.StatusBadGateway, err.Error())
	}
	return dto.NewSuccessResponse(analytics)
}

func (GetUsage) APISettings() feature.APISettings {
	return feature.NewAPISettings("Get WhatsApp template usage", "Gets sent, delivered, and read analytics for WhatsApp message templates.", types.HttpRequestTypeQuery, http.MethodGet, "/v1/wa/templates/usage", true, true, types.APITagWA, []feature.APIError{
		feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
		feature.NewAPIError(*exception.NewCustomException("business portfolio not found", http.StatusNotFound)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
	})
}
