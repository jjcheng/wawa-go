package feature_wa_message

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type UploadMedia struct {
	Filename    string `form:"filename" val:"required" description:"Original media filename" example:"photo.jpeg"`
	ContentType string `form:"content_type" val:"required" description:"Media MIME type" example:"image/jpeg"`
	Content     []byte `json:"-"`
}

func (upload *UploadMedia) Validate() []exception.InputException {
	upload.Filename = filepath.Base(strings.TrimSpace(upload.Filename))
	upload.ContentType = strings.TrimSpace(upload.ContentType)
	inputErrors := []exception.InputException{}
	if upload.Filename == "" || upload.Filename == "." {
		inputErrors = append(inputErrors, exception.NewInputException("filename", "missing filename"))
	}
	if !strings.Contains(upload.ContentType, "/") {
		inputErrors = append(inputErrors, exception.NewInputException("content_type", "invalid content type"))
	}
	if len(upload.Content) == 0 {
		inputErrors = append(inputErrors, exception.NewInputException("body", "media content is required"))
	}
	return inputErrors
}

func (upload UploadMedia) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*Media] {
	if user == nil {
		return dto.NewFailedResponse[*Media](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := upload.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*Media](inputErrors)
	}
	filename := strings.ReplaceAll(uuid.NewString(), "-", "") + filepath.Ext(upload.Filename)
	url, err := dependencies.File.UploadFile(upload.Content, "media", filename, types.StorageClassCool)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, upload.Filename, upload.ContentType)
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewSuccessResponse(&Media{URL: url})
}

func (UploadMedia) APISettings() feature.APISettings {
	return feature.NewBinaryAPISettings(
		"Upload WhatsApp media",
		"Uploads raw media bytes to OSS and returns a permanent URL for use in a WhatsApp media message.",
		types.HttpRequestTypeQuery,
		http.MethodPost,
		"/v1/wa/media",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}

func NewUploadMedia(filename string, contentType string, content []byte) UploadMedia {
	return UploadMedia{Filename: filename, ContentType: contentType, Content: content}
}

func UploadMediaRequestError(err error) dto.Response[*Media] {
	return dto.NewInvalidInputResponse[*Media]([]exception.InputException{exception.NewInputException("body", fmt.Sprintf("failed to read media content: %v", err))})
}
