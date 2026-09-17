package feature_wa_template

import (
	"context"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type UploadExample struct {
	Filename    string `form:"filename" val:"required" description:"Original sample filename" example:"photo.jpeg"`
	ContentType string `form:"content_type" val:"required" description:"Sample MIME type" example:"image/jpeg"`
	Content     []byte `json:"-"`
}

func (upload *UploadExample) Validate() []exception.InputException {
	upload.Filename = filepath.Base(strings.TrimSpace(upload.Filename))
	upload.ContentType = strings.TrimSpace(upload.ContentType)
	inputErrors := []exception.InputException{}
	if upload.Filename == "" || upload.Filename == "." || upload.Filename == ".." || upload.Filename == "/" {
		inputErrors = append(inputErrors, exception.NewInputException("filename", "missing filename"))
	}
	contentType, _, err := mime.ParseMediaType(upload.ContentType)
	if err != nil || !strings.Contains(contentType, "/") {
		inputErrors = append(inputErrors, exception.NewInputException("content_type", "invalid content type"))
	} else {
		upload.ContentType = contentType
	}
	if len(upload.Content) == 0 {
		inputErrors = append(inputErrors, exception.NewInputException("body", "sample content is required"))
	}
	return inputErrors
}

func (upload UploadExample) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*service.WhatsAppTemplateHeaderSampleUploadResponse] {
	if user == nil {
		return dto.NewFailedResponse[*service.WhatsAppTemplateHeaderSampleUploadResponse](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := upload.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*service.WhatsAppTemplateHeaderSampleUploadResponse](inputErrors)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil || strings.TrimSpace(user.WA.BusinessPortfolioAccessToken) == "" {
		return dto.NewFailedResponse[*service.WhatsAppTemplateHeaderSampleUploadResponse](http.StatusUnauthorized, "you are not authorized to access this business portfolio")
	}
	handle, err := dependencies.Whatsapp.UploadTemplateHeaderSample(ctx, upload.Filename, upload.ContentType, upload.Content, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*service.WhatsAppTemplateHeaderSampleUploadResponse](http.StatusBadGateway, err.Error())
	}
	return dto.NewSuccessResponse(&service.WhatsAppTemplateHeaderSampleUploadResponse{Handle: handle})
}

func (UploadExample) APISettings() feature.APISettings {
	return feature.NewBinaryAPISettings(
		"Upload WhatsApp template header sample",
		"Uploads raw sample bytes to Meta and returns a handle for a template header example.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/templates/upload-example",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to access this business portfolio", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
		},
	)
}
