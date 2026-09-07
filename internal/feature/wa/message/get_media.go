package feature_wa_message

import (
	"context"
	"errors"
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
	"gorm.io/gorm"
)

type GetMedia struct {
	MediaID string `uri:"media_id" val:"required" description:"WhatsApp media ID from the inbound webhook"`
}

type Media struct {
	Content     []byte `json:"-"`
	ContentType string `json:"-"`
	Filename    string `json:"-"`
}

func (media Media) BinaryContent() ([]byte, string, string) {
	return media.Content, media.ContentType, media.Filename
}

func (getMedia *GetMedia) Validate() []exception.InputException {
	getMedia.MediaID = strings.TrimSpace(getMedia.MediaID)
	inputErrors := []exception.InputException{}
	if getMedia.MediaID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("media_id", "missing WhatsApp media ID"))
	}
	return inputErrors
}

func (getMedia GetMedia) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*Media] {
	if user == nil {
		return dto.NewFailedResponse[*Media](http.StatusForbidden, "you are not authenticated")
	}
	if inputErrors := getMedia.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*Media](inputErrors)
	}
	phoneNumbers, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id)
	if err != nil {
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if len(phoneNumbers) == 0 {
		return dto.NewFailedResponse[*Media](http.StatusUnauthorized, "you are not authorized to use a WhatsApp phone number")
	}
	businessPortfolio, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioByMetaPhoneNumberId(ctx, phoneNumbers[0].MetaPhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*Media](http.StatusUnauthorized, "you are not authorized to use this WhatsApp phone number")
		}
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	content, contentType, err := dependencies.Whatsapp.DownloadMedia(ctx, getMedia.MediaID, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*Media](http.StatusBadGateway, types.ExceptionMessageBadGateway)
	}
	filename := "whatsapp-media"
	if extensions, err := mime.ExtensionsByType(contentType); err == nil && len(extensions) > 0 {
		filename += extensions[0]
	}
	filename = filepath.Base(filename)
	return dto.NewSuccessResponse(&Media{Content: content, ContentType: contentType, Filename: filename})
}

func (GetMedia) APISettings() feature.APISettings {
	return feature.NewBinaryAPISettings(
		"Get WhatsApp media",
		"Downloads media using its WhatsApp media ID.",
		types.HttpRequestTypeUri,
		http.MethodGet,
		"/v1/wa/media/:media_id",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to use a WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
