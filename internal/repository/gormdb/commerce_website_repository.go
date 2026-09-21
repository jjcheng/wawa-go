package gormdb

import (
	"context"
	"errors"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type CommerceWebsiteRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_commerce.Website]
}

func NewCommerceWebsiteRepository(db *gorm.DB, logger *service.Logger) repository.CommerceWebsiteRepository {
	return &CommerceWebsiteRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_commerce.Website](db, logger),
	}
}

func (websiteRepository *CommerceWebsiteRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_commerce.Website, error) {
	var websites []dao_commerce.Website
	result := websiteRepository.db.WithContext(ctx).
		Table("commerce.websites").
		Joins("JOIN commerce.catalogs ON commerce.catalogs.website_id = commerce.websites.id").
		Where("commerce.websites.business_account_id = ?", businessAccountId).
		Select("commerce.websites.*, commerce.catalogs.name AS catalog_name, commerce.catalogs.meta_id AS meta_catalog_id").
		Order("commerce.websites.id").
		Find(&websites)
	if result.Error != nil {
		websiteRepository.logger.ErrorFunction(result.Error, businessAccountId)
		return nil, result.Error
	}
	return websites, nil
}

func (websiteRepository *CommerceWebsiteRepository) GetByMetaCatalogId(ctx context.Context, metaCatalogId string) (*dao_commerce.Website, error) {
	var website dao_commerce.Website
	result := websiteRepository.db.WithContext(ctx).
		Table("commerce.websites").
		Joins("JOIN commerce.catalogs ON commerce.catalogs.website_id = commerce.websites.id").
		Where("commerce.catalogs.meta_id = ?", metaCatalogId).
		Select("commerce.websites.*, commerce.catalogs.name AS catalog_name, commerce.catalogs.meta_id AS meta_catalog_id").
		First(&website)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			websiteRepository.logger.ErrorFunction(result.Error, metaCatalogId)
		}
		return nil, result.Error
	}
	return &website, nil
}

func (websiteRepository *CommerceWebsiteRepository) GetByDomainName(ctx context.Context, domainName string) (*dao_commerce.Website, error) {
	var website dao_commerce.Website
	result := websiteRepository.db.WithContext(ctx).
		Table("commerce.websites").
		Joins("JOIN commerce.catalogs ON commerce.catalogs.website_id = commerce.websites.id").
		Where("commerce.websites.domain_name = ?", domainName).
		Select("commerce.websites.*, commerce.catalogs.name AS catalog_name, commerce.catalogs.meta_id AS meta_catalog_id").
		First(&website)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			websiteRepository.logger.ErrorFunction(result.Error, domainName)
		}
		return nil, result.Error
	}
	return &website, nil
}
