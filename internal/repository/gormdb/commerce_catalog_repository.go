package gormdb

import (
	"context"
	"errors"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type CommerceCatalogRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_commerce.Catalog]
}

func NewCommerceCatalogRepository(db *gorm.DB, logger *service.Logger) repository.CommerceCatalogRepository {
	return &CommerceCatalogRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_commerce.Catalog](db, logger),
	}
}

func (commerceCatalogRepository *CommerceCatalogRepository) GetByWebsiteId(ctx context.Context, websiteId int32) (*dao_commerce.Catalog, error) {
	var catalog dao_commerce.Catalog
	result := commerceCatalogRepository.db.WithContext(ctx).
		Where("website_id = ?", websiteId).
		First(&catalog)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			commerceCatalogRepository.logger.ErrorFunction(result.Error, websiteId)
		}
		return nil, result.Error
	}
	return &catalog, nil
}

func (commerceCatalogRepository *CommerceCatalogRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_commerce.Catalog, error) {
	var catalogs []dao_commerce.Catalog
	result := commerceCatalogRepository.db.WithContext(ctx).
		Table("commerce.catalogs").
		Joins("JOIN commerce.websites ON commerce.websites.id = commerce.catalogs.website_id").
		Where("commerce.websites.business_account_id = ?", businessAccountId).
		Select("commerce.catalogs.*, commerce.websites.domain_name AS website_domain_name, commerce.websites.status AS website_status, commerce.websites.products_last_synced_at AS website_products_last_synced_at").
		Order("commerce.catalogs.id").
		Find(&catalogs)
	if result.Error != nil {
		commerceCatalogRepository.logger.ErrorFunction(result.Error, businessAccountId)
		return nil, result.Error
	}
	return catalogs, nil
}
