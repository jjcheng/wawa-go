package feature_commerce_website

import (
	"context"
	"errors"
	"net/http"
	"time"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/feature"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type Sync struct {
	Id int32 `uri:"id" val:"required" description:"id of the website"`
}

func (sync *Sync) Validate() []exception.InputException {
	errors := []exception.InputException{}
	if sync.Id <= 0 {
		errors = append(errors, exception.NewInputException("id", "missing id"))
	}
	return errors
}

func (sync Sync) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
	if user == nil {
		return dto.NewFailedResponse[any](http.StatusForbidden, types.ExceptionMessageForbidden, nil)
	}
	if user.WA == nil || user.WA.BusinessPortfolio == nil {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	if errors := sync.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[any](errors)
	}
	// get website
	website, err := dependencies.UnitOfWork.CommerceWebsiteRepository().GetById(ctx, sync.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "website not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if website.BusinessAccountId != user.BusinessAccountId {
		return dto.NewFailedResponse[any](http.StatusUnauthorized, types.ExceptionMessageUnauthorized, nil)
	}
	// get catalog from db
	catalog, err := dependencies.UnitOfWork.CommerceCatalogRepository().GetByWebsiteId(ctx, website.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.NewFailedResponse[any](http.StatusNotFound, "catalog not found", nil)
		}
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// get meta catalog and make sure this catalog exist and belong to the business portfolio
	metaCatalog, err := dependencies.Whatsapp.GetCatalog(ctx, catalog.MetaId, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	// currently only support commerce catalog
	if metaCatalog.Vertical != "commerce" {
		return dto.NewFailedResponse[any](http.StatusNotImplemented, "currently only commerce catalog type is supported", nil)
	}
	// get meta sets
	metaSets, _, err := dependencies.Whatsapp.ListProductSets(ctx, catalog.MetaId, "", "", 999, user.WA.BusinessPortfolioAccessToken)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
	}
	// begin transaction
	transaction := dependencies.UnitOfWork.BeginTransaction()
	var committed bool
	defer func() {
		if !committed {
			transaction.Rollback()
		}
	}()
	// update catalog
	if metaCatalog.Name != catalog.Name {
		if err := transaction.CommerceCatalogRepository().UpdateFields(ctx, catalog.Id, map[string]any{"name": metaCatalog.Name}); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	// get existing sets
	sets, err := dependencies.UnitOfWork.CommerceSetRepository().ListByCatalogId(ctx, catalog.Id)
	if err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// update sets
	for _, metaSet := range metaSets {
		if existingSet := helper.First(sets, func(s dao_commerce.Set) bool {
			return s.MetaId == metaSet.ID
		}); existingSet != nil {
			existingSetIndex := helper.IndexOf(sets, func(s dao_commerce.Set) bool {
				return s.Id == existingSet.Id
			})
			sets[*existingSetIndex].Processed = true
			sets[*existingSetIndex].Name = metaSet.Name
			if metaSet.Name != existingSet.Name {
				if err := transaction.CommerceSetRepository().UpdateFields(ctx, existingSet.Id, map[string]any{"name": metaSet.Name}); err != nil {
					return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
				}
			}
		} else {
			newSet := dao_commerce.Set{
				MetaId:    metaSet.ID,
				Name:      metaSet.Name,
				CatalogId: catalog.Id,
				Processed: true,
			}
			if err := transaction.CommerceSetRepository().Insert(ctx, &newSet); err != nil {
				return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
			sets = append(sets, newSet)
		}
	}
	// unprocessed sets to be deleted
	setsToDelete := helper.Filter(sets, func(s dao_commerce.Set) bool {
		return !s.Processed
	})
	for _, set := range setsToDelete {
		if err := transaction.CommerceSetRepository().DeleteById(ctx, set.Id); err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
	}
	// reload the sets
	sets = helper.Filter(sets, func(s dao_commerce.Set) bool {
		return s.Processed
	})

	// for each set left, process the products
	for _, set := range sets {
		// get meta products
		metaProducts, _, err := dependencies.Whatsapp.ListProductsBySetID(ctx, set.MetaId, "", "", 999, user.WA.BusinessPortfolioAccessToken)
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusBadGateway, err.Error(), err)
		}
		// get products
		products, _, _, err := dependencies.UnitOfWork.CommerceGenericProductRepository().List(ctx, set.CatalogId, set.Id, "", "", false, 1, 999)
		if err != nil {
			return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		for _, metaProduct := range metaProducts {
			if existingProduct := helper.First(products, func(p dao_commerce.GenericProduct) bool {
				return p.MetaId == metaProduct.ID
			}); existingProduct != nil {
				existingProduct.Name = metaProduct.Name
				existingProduct.Price = metaProduct.Price
				existingProduct.SalePrice = metaProduct.SalePrice
				existingProduct.Availability = metaProduct.Availability
				existingProduct.ImageUrl = metaProduct.ImageURL
				existingProduct.AdditionalImageUrls = metaProduct.AdditionalImageUrls
				existingProduct.Condition = metaProduct.Condition
				existingProduct.Currency = metaProduct.Currency
				existingProduct.Description = metaProduct.Description
				existingProduct.Gender = metaProduct.Gender
				existingProduct.FBCategory = metaProduct.FBCategory
				existingProduct.RetailerId = metaProduct.RetailerID
				existingProduct.Url = metaProduct.URL
				existingProduct.Processed = true
				if err := transaction.CommerceGenericProductRepository().Update(ctx, existingProduct); err != nil {
					return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
				}
			} else {
				newProduct := dao_commerce.GenericProduct{
					SetId:               set.Id,
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
					Processed:           true,
				}
				if err = transaction.CommerceGenericProductRepository().Insert(ctx, &newProduct); err != nil {
					if errors.Is(err, gorm.ErrDuplicatedKey) {
						continue
					}
					return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
				}
			}
		}
		// delete any unprocessed
		productsToDelete := helper.Filter(products, func(p dao_commerce.GenericProduct) bool {
			return !p.Processed
		})
		for _, product := range productsToDelete {
			if err := transaction.CommerceGenericProductRepository().DeleteById(ctx, product.Id); err != nil {
				return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
			}
		}
	}
	// update products last synced at
	website.ProductsLastSynedAt = time.Now()
	if err := transaction.CommerceWebsiteRepository().Update(ctx, website); err != nil {
		// no need to return error
		dependencies.Logger.ErrorFunction(err, website.Id)
	}
	// commit
	if err := transaction.CommitTransaction(); err != nil {
		return dto.NewFailedResponse[any](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	committed = true
	return dto.NewEmptyResponse(true, http.StatusOK)
}

func (Sync) APISettings() feature.APISettings {
	return feature.NewAPISettings(
		"Sync commerce website",
		"Syncs the catalog, sets and products for a website belonging to the authenticated user's business account.",
		types.HttpRequestTypeUri,
		http.MethodPost,
		"/v1/commerce/websites/:id/sync",
		true,
		true,
		types.APITagCommerce,
		[]feature.APIError{
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageForbidden, http.StatusForbidden)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageUnauthorized, http.StatusUnauthorized)),
			feature.NewAPIError(*exception.NewCustomException("website not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException("catalog not found", http.StatusNotFound)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageBadGateway, http.StatusBadGateway)),
			feature.NewAPIError(*exception.NewCustomException(types.ExceptionMessageInternalServerError, http.StatusInternalServerError)),
		},
	)
}
