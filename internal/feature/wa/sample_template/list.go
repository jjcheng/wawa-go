package feature_wa_sample_template

import (
	"context"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type List struct {
	Category types.WATemplateCategory `form:"category" description:"category of the templates"`
	Language string                   `form:"language" description:"language of the templates"`
	Page     int                      `form:"page" descrption:"page number from 1"`
	PageSize int                      `form:"page_size" description:"number of templates per page"`
}

func (list *List) Validate() []exception.InputException {
	list.Language = strings.TrimSpace(list.Language)
	if list.Page <= 0 {
		list.Page = 1
	}
	if list.PageSize <= 0 {
		list.PageSize = 25
	}
	inputErrors := []exception.InputException{}
	return inputErrors
}

func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_wa.SampleTemplate] {
	if errors := list.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[[]dto_wa.SampleTemplate](errors)
	}
	templates, err := dependencies.UnitOfWork.WASampleTemplateRepository().List(list.Category, list.Language)
	if err != nil {
		return dto.NewFailedResponse[[]dto_wa.SampleTemplate](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	result := make([]dto_wa.SampleTemplate, 0, len(templates))
	for _, template := range templates {
		dtoTemplate, err := dto_wa.NewSampleTemplate(template)
		if err != nil {
			return dto.NewFailedResponse[[]dto_wa.SampleTemplate](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
		}
		result = append(result, *dtoTemplate)
	}

	return dto.NewSuccessResponse(result)
}

func (List) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"List WhatsApp sample templates",
		"Lists locally stored WhatsApp message template samples for the authenticated user.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/sample-templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
