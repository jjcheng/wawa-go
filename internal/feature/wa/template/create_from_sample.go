package feature_wa_template

import (
	"context"
	"errors"
	"fmt"
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

type CreateFromSample struct {
	SampleTemplateId int    `json:"sample_template_id" val:"required" description:"id of the sample template"`
	Name             string `json:"name" val:"required" description:"name of the new template"`
}

func (createFromSample *CreateFromSample) Validate() []exception.InputException {
	createFromSample.Name = strings.TrimSpace(createFromSample.Name)
	inputErrors := []exception.InputException{}
	if createFromSample.SampleTemplateId <= 0 {
		inputErrors = append(inputErrors, exception.NewInputException("sample_template_id", "missing sample template id"))
	}
	if createFromSample.Name == "" {
		inputErrors = append(inputErrors, exception.NewInputException("name", "missing template name"))
	} else {
		createFromSample.Name = strings.ReplaceAll(createFromSample.Name, " ", "_")
	}
	return inputErrors
}

func (createFromSample CreateFromSample) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_wa.Template] {
	if inputErrors := createFromSample.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_wa.Template](inputErrors)
	}
	businessPortfolio, businessAccount, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().GetBusinessPortfolioAndAccountByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusUnauthorized, "you are not authorized to access this WABA")
		}
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	sampleTemplate, err := dependencies.UnitOfWork.WASampleTemplateRepository().GetById(ctx, int32(createFromSample.SampleTemplateId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_wa.Template](http.StatusNotFound, "sample template not found")
		}
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	templateBase, err := dto_wa.NewSampleTemplate(*sampleTemplate)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	templateBase.Name = createFromSample.Name
	if err := createFromSample.uploadHeaderMediaSamples(ctx, &templateBase.TemplateBase, dependencies, businessPortfolio.AccessToken); err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	template, err := dependencies.Whatsapp.CreateTemplate(ctx, businessAccount.MetaWABAId, templateBase.Payload(), businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*dto_wa.Template](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	// set meta edit template url
	template.MetaEditTemplateUrl = fmt.Sprintf("https://business.facebook.com/latest/whatsapp_manager/message_templates/?business_id=%s&tab=message-templates&childRoute=CAPI&id=%s&nav_ref=whatsapp_manager&asset_id=%s", businessPortfolio.MetaBusinessPortfolioId, template.ID, businessAccount.MetaWABAId)
	return dto.NewSuccessResponse(template)
}

func (createFromSample CreateFromSample) uploadHeaderMediaSamples(ctx context.Context, templateBase *dto_wa.TemplateBase, dependencies *service.Dependencies, businessAccessToken string) error {
	for componentIndex := range templateBase.Components {
		component := &templateBase.Components[componentIndex]
		if component.Type != types.WATemplateComponentTypeHeader || !isTemplateHeaderMediaFormat(component.Format) {
			continue
		}
		if component.Example == nil || len(component.Example.HeaderHandle) == 0 {
			return errors.New("missing header media sample")
		}
		mediaURL := strings.TrimSpace(component.Example.HeaderHandle[0])
		content, contentType, filename, err := dependencies.Whatsapp.DownloadFile(ctx, mediaURL, businessAccessToken)
		if err != nil {
			return err
		}
		mediaHandle, err := dependencies.Whatsapp.UploadTemplateHeaderSample(ctx, filename, contentType, content, businessAccessToken)
		if err != nil {
			return err
		}
		component.Example.HeaderHandle[0] = mediaHandle
	}
	return nil
}

func isTemplateHeaderMediaFormat(format types.WATemplateComponentFormat) bool {
	switch format {
	case types.WATemplateComponentFormatImage, types.WATemplateComponentFormatVideo, types.WATemplateComponentFormatDocument:
		return true
	default:
		return false
	}
}

func (CreateFromSample) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create WhatsApp template from sample",
		"Creates a WhatsApp message template from a locally stored sample template.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/wa/templates/from-sample",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this WABA", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("sample template not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
