package feature_commerce_website

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
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

type Create struct {
	Subdomain     string `json:"subdomain" val:"required" description:"subdomain name of the website"`
	MetaCatalogId string `json:"meta_catalog_id" val:"required" description:"Meta catalog id to create website from"`
}

func (create *Create) Validate() []exception.InputException {
	create.Subdomain = strings.ToLower(strings.TrimSpace(create.Subdomain))
	create.MetaCatalogId = strings.TrimSpace(create.MetaCatalogId)
	inputErrors := []exception.InputException{}
	if create.Subdomain == "" {
		inputErrors = append(inputErrors, exception.NewInputException("subdomain", "missing subdomain"))
	} else if len(create.Subdomain) > 60 || !regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`).MatchString(create.Subdomain) {
		inputErrors = append(inputErrors, exception.NewInputException("subdomain", "invalid subdomain"))
	} else {
		reservedSubdomains := []string{"admin", "portal", "hello", "public", "private", "reserved", "panel", "api", "web", "app", "website", "application", "gateway", "gate", "mobile", "mobile-web", "mobileweb", "system", "llm", "model", "open", "closed", "wwww", "wwwww", "ww", "www", "xyz"}
		if helper.Any(reservedSubdomains, func(s string) bool {
			return s == create.Subdomain
		}) {
			inputErrors = append(inputErrors, exception.NewInputException("subdomain", "this subdomain is reserved"))
		}
	}
	if create.MetaCatalogId == "" {
		inputErrors = append(inputErrors, exception.NewInputException("meta_catalog_id", "missing Meta catalog id"))
	}
	return inputErrors
}

func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_commerce.Website] {
	if user == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusForbidden, "you are not authenticated")
	}
	if user.Type != types.UserTypeMaster || user.WA == nil || user.WA.BusinessAccount == nil || user.WA.PhoneNumber_ == nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusUnauthorized, "you are not authorized")
	}
	if inputErrors := create.Validate(); len(inputErrors) > 0 {
		return dto.NewInvalidInputResponse[*dto_commerce.Website](inputErrors)
	}
	// make sure no existing website for this catalog
	existingWebsite, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetByMetaCatalogId(ctx, create.MetaCatalogId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if existingWebsite != nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusBadRequest, "there is already an existing website for this catalog")
	}
	// check existing domain name
	existingWebsite, err = dependencies.UnitOfWork.CommerceWebsiteRepository().GetByDomainName(ctx, create.Subdomain)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	if existingWebsite != nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusConflict, "subdomain is already in use")
	}
	// get metaCatalog and make sure this catalog exist and belong to the business portfolio
	metaCatalog, err := dependencies.Whatsapp.GetCatalog(ctx, create.MetaCatalogId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, create.MetaCatalogId)
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusBadGateway, err.Error())
	}
	// currently only support commerce catalog
	if metaCatalog.Vertical != "commerce" {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusNotImplemented, "currently only commerce catalog type is supported")
	}
	// get phone number business profile
	phoneNumberBusinessProfile, err := dependencies.Whatsapp.GetPhoneNumberBusinessProfile(ctx, user.WA.PhoneNumber_.MetaPhoneNumberId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, user.WA.PhoneNumber_)
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusBadGateway, err.Error())
	}
	// get metaSets
	metaSets, _, err := dependencies.Whatsapp.ListProductSets(ctx, create.MetaCatalogId, "", "", 999, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, create.MetaCatalogId)
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusBadGateway, err.Error())
	}
	// get products for each set
	var metaProducts [][]service.WhatsAppProduct
	for _, metaSet := range metaSets {
		products, _, err := dependencies.Whatsapp.ListProductsBySetID(ctx, metaSet.ID, "", "", 999, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, metaSet.ID)
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusBadGateway, err.Error())
		}
		metaProducts = append(metaProducts, products)
	}
	// start transaction
	var committed bool
	transaction := dependencies.UnitOfWork.BeginTransaction()
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	// create website
	website := dao_commerce.Website{
		BusinessAccountId:   user.WA.BusinessAccount.Id,
		DomainName:          create.Subdomain,
		Status:              types.CommerceWebsiteStatusActive,
		ProductsLastSynedAt: time.Now(),
	}
	if phoneNumberBusinessProfile != nil && len(phoneNumberBusinessProfile.Data) > 0 {
		website.About = phoneNumberBusinessProfile.Data[0].About
		website.Description = phoneNumberBusinessProfile.Data[0].Description
		website.ProfilePictureURL = phoneNumberBusinessProfile.Data[0].ProfilePictureURL
		website.Address = phoneNumberBusinessProfile.Data[0].Address
		website.Email = phoneNumberBusinessProfile.Data[0].Email
		website.Vertical = phoneNumberBusinessProfile.Data[0].Vertical
	}
	if err := transaction.CommerceWebsiteRepository().Insert(ctx, &website); err != nil {
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, types.ExceptionMessageInternalServerError)
	}
	// create a catalog
	newCatalog := dao_commerce.Catalog{
		WebsiteId: website.Id,
		MetaId:    create.MetaCatalogId,
		Name:      metaCatalog.Name,
		Vertical:  metaCatalog.Vertical,
	}
	err = transaction.CommerceCatalogRepository().Insert(ctx, &newCatalog)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, newCatalog)
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, "failed to create catalog")
	}
	// create sets
	for i, metaSet := range metaSets {
		// insert set
		newSet := dao_commerce.Set{
			CatalogId: newCatalog.Id,
			MetaId:    metaSet.ID,
			Name:      metaSet.Name,
		}
		err = transaction.CommerceSetRepository().Insert(ctx, &newSet)
		if err != nil {
			dependencies.Logger.ErrorFunction(err, newSet)
			return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, "failed to create set")
		}
		// insert products
		for _, metaProduct := range metaProducts[i] {
			newProduct := dao_commerce.GenericProduct{
				SetId:               newSet.Id,
				MetaId:              metaProduct.ID,
				Name:                metaProduct.Name,
				Price:               metaProduct.Price,
				SalePrice:           metaProduct.SalePrice,
				Availability:        metaProduct.Availability,
				ImageUrl:            metaProduct.ImageURL,
				AdditionalImageUrls: metaProduct.AdditionalImageUrls,
				Condition:           metaProduct.Condition,
				Currency:            metaProduct.Currency,
				Description:         metaProduct.Description,
				Gender:              metaProduct.Gender,
				FBCategory:          metaProduct.FBCategory,
				RetailerId:          metaProduct.RetailerID,
				Url:                 metaProduct.URL,
			}
			err = transaction.CommerceGenericProductRepository().Insert(ctx, &newProduct)
			if err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					continue
				}
				dependencies.Logger.ErrorFunction(err, newProduct)
				return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, "failed to create product")
			}
		}
	}
	if err := transaction.CommitTransaction(); err != nil {
		dependencies.Logger.ErrorFunction(err)
		return dto.NewFailedResponse[*dto_commerce.Website](http.StatusInternalServerError, "failed to commit transaction")
	}
	committed = true
	result := dto_commerce.NewWebsite(website)
	result.CatalogName = metaCatalog.Name
	result.MetaCatalogId = create.MetaCatalogId
	return dto.NewSuccessResponse(&result)
}

func (Create) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Create commerce website",
		"Creates a website for a catalog belonging to the authenticated user's business account.",
		types.HttpRequestTypeJSON,
		http.MethodPost,
		"/v1/commerce/websites",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException("you are not authenticated", http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException("you are not authorized", http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("catalog not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("website already exists", http.StatusConflict)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
