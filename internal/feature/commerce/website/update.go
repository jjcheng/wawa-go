package feature_commerce_website

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Update struct {
	Id                int32  `uri:"id" val:"required" description:"id of the website"`
	About             string `json:"about"`
	Description       string `json:"description"`
	ProfilePictureURL string `json:"profile_picture_url"`
	Address           string `json:"address"`
	Email             string `json:"email"`
	// interal settings
	ContactText   string  `json:"contact_text"`
	CoverImageURL string  `json:"cover_image_url"`
	Tagline       string  `json:"tagline"`
	Latitude      float32 `json:"latitude"`
	Longitude     float32 `json:"longitude"`
	CopyrightText string  `json:"copyright_text"`
}

func (update *Update) Validate() []exception.InputException {
	update.About = strings.TrimSpace(update.About)
	update.Description = strings.TrimSpace(update.Description)
	update.ProfilePictureURL = strings.TrimSpace(update.ProfilePictureURL)
	update.Address = strings.TrimSpace(update.Address)
	update.Email = strings.TrimSpace(update.Email)
	update.ContactText = strings.TrimSpace(update.ContactText)
	update.CoverImageURL = strings.TrimSpace(update.CoverImageURL)
	update.Tagline = strings.TrimSpace(update.Tagline)
	update.CopyrightText = strings.TrimSpace(update.CopyrightText)
	var errors []exception.InputException
	if update.Id <= 0 {
		errors = append(errors, exception.NewInputException("id", "invalid website id"))
	}
	if update.Email != "" && !helper.ValidateEmail(update.Email) {
		errors = append(errors, exception.NewInputException("email", "invalid email"))
	}
	if update.About == "" {
		errors = append(errors, exception.NewInputException("about", "missing about"))
	}
	if update.Description == "" {
		errors = append(errors, exception.NewInputException("description", "missing description"))
	}
	return nil
}

func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Website] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.Type != types.UserTypeMaster {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if inputErrors := update.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Website](inputErrors)
	}
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, update.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	website.About = update.About
	website.Description = update.Description
	website.ProfilePictureURL = update.ProfilePictureURL
	website.Address = update.Address
	website.Email = update.Email
	website.ContactText = update.ContactText
	website.Tagline = update.Tagline
	website.CoverImageUrl = update.CoverImageURL
	website.Latitude = update.Latitude
	website.Longitude = update.Longitude
	website.CopyrightText = update.CopyrightText
	if err := dependencies.UnitOfWork.CommerceWebsiteRepository().Update(ctx, website); err != nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	result := dto_commerce.NewWebsite(*website)
	return dto.NewSuccessResponse(&result)
}

func (Update) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Update commerce website",
		"Updates the profile information of a website belonging to the authenticated user's business account.",
		types.HttpRequestTypeUriJSON,
		http.MethodPatch,
		"/v1/commerce/websites/:id",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
