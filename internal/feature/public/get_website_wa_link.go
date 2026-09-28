package feature_public

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
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

// do a rotation of user to be featured as the whatsapp contact
type GetWebsiteWALink struct {
}

func (getWebsiteWALink *GetWebsiteWALink) Validate() []exception.InputException {
	return nil
}

func (getWebsiteWALink GetWebsiteWALink) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
	if errors := getWebsiteWALink.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}
	website := getWebsiteFromContext(ctx)
	if website == nil {
		return dto.NewFailedResponse[string](http.StatusNotFound, "website not found", nil)
	}
	businessAccount, err := dependencies.UnitOfWork.WABusinessAccountRepository().GetById(ctx, website.BusinessAccountId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[string](http.StatusNotFound, "business account not found", nil)
		}
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// get cached website last whatsapp contact
	cacheKey := helper.GetWebsiteLastWhatsAppContactUserIdCacheKey(website.Id)
	cache, err := dependencies.Cache.Get(ctx, cacheKey)
	if err != nil {
		if !errors.Is(err, service.CacheNotFoundError) {
			// don't return error
			dependencies.Logger.Error(err)
		}
	}
	var user dao_account.User
	if cache == "" { // if no cache, use the first active user
		u, err := getWebsiteWALink.getFirstActiveUser(ctx, businessAccount.Id, dependencies)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[string](http.StatusNotFound, "no active user", nil)
			}
			return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		user = *u
	} else {
		// get user id
		dic, err := helper.DeserializeJSON[map[string]string](cache)
		if err != nil {
			return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		userId, err := strconv.Atoi((*dic)["value"])
		if err != nil {
			dependencies.Logger.ErrorFunction(err, cache)
		}
		nextActiveUser, err := getWebsiteWALink.getNextActiveUser(ctx, businessAccount.Id, int32(userId), dependencies)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.NewFailedResponse[string](http.StatusNotFound, "no next active user", nil)
			}
			return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		user = *nextActiveUser
	}
	// set cache key for 1 year, ignore error
	_ = dependencies.Cache.Set(ctx, cacheKey, fmt.Sprint(user.Id), 30*24*360)
	// get phone number
	phoneNumbers, _, _, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByUserId(ctx, user.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[string](http.StatusNotFound, "no phone number found", nil)
		}
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	phoneNumberDTO := dto_wa.NewPhoneNumber(phoneNumbers[0])
	waLink := phoneNumberDTO.WALink(website.ContactText)
	return dto.NewSuccessResponse(waLink)
}

func (getWebsiteWALink *GetWebsiteWALink) getFirstActiveUser(ctx context.Context, businessAccountId int32, dependencies *service.Dependencies) (*dao_account.User, error) {
	allUsers, err := dependencies.UnitOfWork.AccountUserRepository().ListByBusinessAccountId(ctx, businessAccountId, nil)
	if err != nil {
		return nil, fmt.Errorf("GetWebsiteWALink.getFirstActiveUser businessAccountId=%d error=%w", businessAccountId, err)
	}
	activeUsers := helper.Filter(allUsers, func(u dao_account.User) bool {
		return u.Status == types.UserStatusActive
	})
	if len(activeUsers) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	// if only 1 active user, use him
	if len(activeUsers) == 1 {
		return &activeUsers[0], nil
	}
	helper.Sort(activeUsers, func(a, b dao_account.User) int {
		if a.Id < b.Id {
			return -1
		} else if a.Id > b.Id {
			return 1
		}
		return 0
	})
	user := activeUsers[0]
	return &user, nil
}

func (getWebsiteWALink *GetWebsiteWALink) getNextActiveUser(ctx context.Context, businessAccountId int32, currentUserId int32, dependencies *service.Dependencies) (*dao_account.User, error) {
	allUsers, err := dependencies.UnitOfWork.AccountUserRepository().ListByBusinessAccountId(ctx, businessAccountId, nil)
	if err != nil {
		return nil, fmt.Errorf("GetWebsiteWALink.getNextActiveUser businessAccountId=%d currentUserId=%d error=%w", businessAccountId, currentUserId, err)
	}
	activeUsers := helper.Filter(allUsers, func(u dao_account.User) bool {
		return u.Status == types.UserStatusActive
	})
	if len(activeUsers) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	// if only 1 active user, use him
	if len(activeUsers) == 1 {
		return &activeUsers[0], nil
	}
	helper.Sort(activeUsers, func(a, b dao_account.User) int {
		if a.Id < b.Id {
			return -1
		} else if a.Id > b.Id {
			return 1
		}
		return 0
	})
	for i := range activeUsers {
		if activeUsers[i].Id != currentUserId {
			continue
		}
		if i == len(activeUsers)-1 { // if current user is the last, take first
			return &activeUsers[0], nil
		} else {
			return &activeUsers[i+1], nil
		}
	}
	// it's possible that the userId is invalid, just take the first user
	return &activeUsers[0], nil
}

func (GetWebsiteWALink) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Get WhatsApp link of the website",
		"Use rotation strategy to get the next user's WhatsApp link for the website.",
		types.HttpRequestTypeNone,
		http.MethodGet,
		"/v1/public/wa-link",
		false,
		true,
		types.APITagPublic,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
