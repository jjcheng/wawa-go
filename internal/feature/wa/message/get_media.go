package feature_wa_message

import (
	"context"
	"errors"
	"net/http"
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
	WAMessageId string `form:"wa_message_id" val:"required" description:"WhatsApp message ID"`
	WAMediaID   string `form:"wa_media_id" val:"required" description:"WhatsApp media ID"`
}

type Media struct {
	ID  string `json:"id,omitempty" description:"if to_meta is true, will return media ID"`
	URL string `json:"url" description:"Permanent OSS URL for the WhatsApp media"`
}

func (getMedia *GetMedia) Validate() []exception.InputException {
	getMedia.WAMediaID = strings.TrimSpace(getMedia.WAMediaID)
	getMedia.WAMessageId = strings.TrimSpace(getMedia.WAMessageId)
	inputErrors := []exception.InputException{}
	if getMedia.WAMediaID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("wa_media_id", "missing WA media id"))
	}
	if getMedia.WAMessageId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("wa_message_id", "missing WA message id"))
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
	message, err := dependencies.UnitOfWork.WAMessageRepository().GetByWAMessageId(ctx, getMedia.WAMessageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*Media](http.StatusNotFound, "message not found")
		}
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if !payloadContainsMediaID(message.Payload, getMedia.WAMediaID) {
		return dto.NewFailedResponse[*Media](http.StatusNotFound, "media not found in message")
	}
	phoneNumbers, _, _, err := dependencies.UnitOfWork.WAUserPhoneNumberRepository().ListPhoneNumbersByUserId(ctx, user.Id, 1, 999)
	if err != nil {
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	authorized := false
	for _, phoneNumber := range phoneNumbers {
		if phoneNumber.MetaPhoneNumberId == message.PhoneNumberId {
			authorized = true
			break
		}
	}
	if !authorized {
		return dto.NewFailedResponse[*Media](http.StatusUnauthorized, "you are not authorized to access this message")
	}
	businessPortfolio, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetBusinessPortfolioByMetaPhoneNumberId(ctx, message.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*Media](http.StatusUnauthorized, "you are not authorized to use this phone number")
		}
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	content, contentType, err := dependencies.Whatsapp.DownloadMedia(ctx, getMedia.WAMediaID, businessPortfolio.AccessToken)
	if err != nil {
		return dto.NewFailedResponse[*Media](http.StatusBadGateway, err.Error())
	}
	filename := getMedia.WAMediaID
	if contentTypeParts := strings.SplitN(contentType, "/", 2); len(contentTypeParts) == 2 && contentTypeParts[1] != "" {
		filename += "." + contentTypeParts[1]
	}
	attachmentURL, err := dependencies.File.UploadFile(content, "media", filename, types.StorageClassCool)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, getMedia.WAMediaID)
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	message.AttachmentURL = attachmentURL
	if err := dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message); err != nil {
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	return dto.NewSuccessResponse(&Media{URL: attachmentURL})
}

func payloadContainsMediaID(payload map[string]any, mediaID string) bool {
	for _, mediaType := range []string{"audio", "document", "image", "sticker", "video"} {
		media, ok := payload[mediaType].(map[string]any)
		if !ok {
			continue
		}
		id, _ := media["id"].(string)
		if id == mediaID {
			return true
		}
	}
	return false
}

func (GetMedia) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp media",
		"Downloads WhatsApp media, persists it and returns its permanent URL.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/media",
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
