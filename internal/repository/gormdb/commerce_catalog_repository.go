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
