package feature_wa_message

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type GetMedia struct {
	MessageId int32  `form:"message_id" val:"required" description:"id of the message"`
	WAMediaID string `form:"wa_media_id" val:"required" description:"WhatsApp media ID"`
}

type Media struct {
	ID  string `json:"id,omitempty" description:"if to_meta is true, will return media ID"`
	URL string `json:"url" description:"Permanent OSS URL for the WhatsApp media"`
}

func (getMedia *GetMedia) Validate() []exception.InputException {
	getMedia.WAMediaID = strings.TrimSpace(getMedia.WAMediaID)
	inputErrors := []exception.InputException{}
	if getMedia.WAMediaID == "" {
		inputErrors = append(inputErrors, exception.NewInputException("wa_media_id", "missing WA media id"))
	}
	if getMedia.MessageId < 0 {
		inputErrors = append(inputErrors, exception.NewInputException("message_id", "missing message id"))
	}
	return inputErrors
}

func (getMedia GetMedia) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*Media] {
	if user == nil {
		return dto.NewFailedResponse[*Media](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil {
		return dto.NewFailedResponse[*Media](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := getMedia.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*Media](inputErrors)
	}
	message, err := dependencies.UnitOfWork.WAMessageRepository().GetById(ctx, getMedia.MessageId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*Media](http.StatusNotFound, "message not found", nil)
		}
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// check message belongs to user
	phoneNumber, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetById(ctx, message.PhoneNumberId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*Media](http.StatusNotFound, "phone number not found", nil)
		}
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if !helper.Any(user.WA.PhoneNumbers, func(pn dto_wa.PhoneNumber) bool {
		return pn.Id == phoneNumber.Id
	}) {
		return dto.NewFailedResponse[*Media](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// download from Meta
	content, contentType, err := dependencies.Whatsapp.DownloadMedia(ctx, getMedia.WAMediaID, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[*Media](http.StatusBadGateway, err.Error(), err)
	}
	filename := getMedia.WAMediaID
	if contentTypeParts := strings.SplitN(contentType, "/", 2); len(contentTypeParts) == 2 && contentTypeParts[1] != "" {
		filename += "." + contentTypeParts[1]
	}
	// upload to OSS
	attachmentURL, err := dependencies.File.UploadFile(content, "media", filename, types.StorageClassCool)
	if err != nil {
		return dto.NewFailedResponse[*Media](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	message.AttachmentURL = attachmentURL
	// ignore error
	_ = dependencies.UnitOfWork.WAMessageRepository().Update(ctx, message)
	return dto.NewSuccessResponse(&Media{URL: attachmentURL})
}

func (GetMedia) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp media",
		"Download WhatsApp media, persists it and returns its permanent URL.",
		types.HttpRequestTypeQuery,
		http.MethodGet,
		"/v1/wa/media",
		true,
		true,
		types.APITagWA,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized to use a WhatsApp phone number", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
