package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_public "github.com/jjcheng/wawa-go/internal/feature/public"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

func Authenticate(dependencies *service.Dependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userAccessToken := ctx.GetHeader(cfg.Default().Site.HTTPHeaderUserAccessTokenKey)
		if userAccessToken != "" {
			tokenHash := helper.HashSHA256Hex(userAccessToken)
			if dependencies.AuthCache != nil {
				if cachedUser, ok := dependencies.AuthCache.Get(tokenHash, time.Now()); ok {
					dependencies.UnitOfWork.AccountSessionRepository().UpdateLastUsed(ctx, cachedUser.Session.Id)
					ctx.Set(cfg.Default().Site.HTTPRequestUserKey, cachedUser)
					ctx.Next()
					return
				}
			}
			// get session
			session, err := dependencies.UnitOfWork.AccountSessionRepository().GetByAccessTokenHash(ctx.Request.Context(), tokenHash)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
					ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
					return
				}
				responseObject := dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			// if expired, return
			if !session.ExpiresAt.After(time.Now()) {
				responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "session expired, please login again", nil)
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			// if revoked, return
			if session.RevokedAt != nil {
				responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "session revoked, please login again", nil)
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			// update last used
			dependencies.UnitOfWork.AccountSessionRepository().UpdateLastUsed(ctx, session.Id)
			// get user
			user, err := dependencies.UnitOfWork.AccountUserRepository().GetById(ctx.Request.Context(), session.UserId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
					ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
					return
				}
				responseObject := dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			// check user status
			if user.Status == types.UserStatusInactive || user.Status == types.UserStatusClosed {
				responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
				return
			}
			userDTO := dto_account.NewUser(*user)
			userDTO.Session = helper.ConvertToPointer(dto_account.NewSession(*session))
			// load wa reloated objects
			phoneNumbers, businessAccount, businessPortfolio, err := dependencies.UnitOfWork.WAPhoneNumberRepository().GetByUserId(ctx.Request.Context(), user.Id)
			if err == nil && len(phoneNumbers) > 0 {
				managedPhoneNumbers := make([]dto_wa.PhoneNumber, 0, len(phoneNumbers))
				for _, phoneNumber := range phoneNumbers {
					managedPhoneNumbers = append(managedPhoneNumbers, dto_wa.NewPhoneNumber(phoneNumber))
				}
				businessAccountDTO := dto_wa.NewBusinessAccount(*businessAccount, businessPortfolio.MetaBusinessPortfolioId, businessPortfolio.Name)
				businessPortfolioDTO := dto_wa.NewBusinessPortfolio(*businessPortfolio, false)
				userDTO.WA = &dto_account.UserWA{
					PhoneNumbers:                 managedPhoneNumbers,
					BusinessAccount:              &businessAccountDTO,
					BusinessPortfolio:            &businessPortfolioDTO,
					BusinessPortfolioAccessToken: businessPortfolio.AccessToken,
				}
			}
			if dependencies.AuthCache != nil {
				dependencies.AuthCache.Set(tokenHash, &userDTO, time.Now())
			}
			ctx.Set(cfg.Default().Site.HTTPRequestUserKey, &userDTO)
		} else { // public website
			origin := ctx.GetHeader("X-Forwarded-Host")
			if origin == "" {
				origin = ctx.GetHeader("Host")
			}
			if origin != "" {
				getWebsiteByDomain := feature_public.GetWebsiteByDomain{
					Domain: origin,
				}
				getWebsiteByDomainResponse := getWebsiteByDomain.Handle(ctx, nil, dependencies)
				if getWebsiteByDomainResponse.Success {
					ctx.Set(cfg.Default().Site.HTTPRequestWebsiteKey, getWebsiteByDomainResponse.Data)
				}
			}
		}
		ctx.Next()
	}
}
