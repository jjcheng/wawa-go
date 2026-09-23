package feature_wa_sample_template

import (
	"context"
	"encoding/json"
	"net/http"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

// this is only for WAWA admin
type Create struct {
	dto_wa.TemplateBase
}

func (create *Create) Validate() []exception.InputException {
	return create.TemplateBase.Validate()
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.SampleTemplate] {
	// TODO: only WAWA admin can call this
	if user == nil {
		return dto.NewFailedResponse[*dto_wa.SampleTemplate](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_wa.SampleTemplate](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := create.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.SampleTemplate](errors)
	}
	daoTemplate := dao_wa.SampleTemplate{
		Name:            create.Name,
		Category:        create.Category,
		Languauge:       create.Language,
		ParameterFormat: create.ParameterFormat,
		Components:      make([]map[string]any, 0, len(create.Components)),
	}
	payload, err := json.Marshal(create.Components)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.SampleTemplate](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if err := json.Unmarshal(payload, &daoTemplate.Components); err != nil {
		return dto.NewFailedResponse[*dto_wa.SampleTemplate](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if err := dependencies.UnitOfWork.WASampleTemplateRepository().Insert(ctx, &daoTemplate); err != nil {
		return dto.NewFailedResponse[*dto_wa.SampleTemplate](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result, err := dto_wa.NewSampleTemplate(daoTemplate)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.SampleTemplate](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	return dto.NewSuccessResponse(result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp sample template",
		"Creates a locally stored WhatsApp template sample used by your app.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/sample-templates",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
