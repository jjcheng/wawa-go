package feature_public

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_commerce "github.com/jjcheng/wawa-go/internal/dto/commerce"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

// this is not a public API
type GetWebsiteByDomain struct {
	Domain string `form:"domain" val:"required" description:"domain name of the website"`
}

func (getWebsiteByDomain *GetWebsiteByDomain) Validate() []exception.InputException {
	getWebsiteByDomain.Domain = strings.TrimSpace(getWebsiteByDomain.Domain)
	if getWebsiteByDomain.Domain == "" {
		return []exception.InputException{exception.NewInputException("domain", "missing domain")}
	}
	return nil
}

func (getWebsiteByDomain GetWebsiteByDomain) Handle(ctx context.Context, _ *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Website] {
	if inputErrors := getWebsiteByDomain.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Website](inputErrors)
	}
	// the domain passed in is the full host like beccas-cafe.wawago.app, trim the end commerceWebsiteDomain first
	domain := strings.TrimSuffix(getWebsiteByDomain.Domain, cfg.Default().Commerce.WebsiteDomain)
	domain = strings.TrimSuffix(domain, ".")
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetByDomainName(ctx, domain)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			dependencies.Logger.ErrorFunction(err, getWebsiteByDomain.Domain)
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, "failed to get website by subdomain")
		}
	}
	if website == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "website not found")
	}
	if website.Status == types.CommerceWebsiteStatusInactive {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotFound, "website not found")
	}
	result := dto_commerce.NewWebsite(*website)
	// hide some sensitive fields
	result.MetaCatalogId = ""
	return dto.NewSuccessResponse(&result)
}

func getWebsiteFromContext(ctx context.Context) *dto_commerce.Website {
	var website *dto_commerce.Website
	ginCtx := helper.GetGinContext(ctx)
	v, exist := ginCtx.Get(cfg.Default().Site.HTTPRequestWebsiteKey)
	if exist {
		website = v.(*dto_commerce.Website)
		return website
	}
	return nil
}
