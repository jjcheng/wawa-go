package gormdb

import (
	"context"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type CommerceSetRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_commerce.Set]
}

func NewCommerceSetRepository(db *gorm.DB, logger *service.Logger) repository.CommerceSetRepository {
	return &CommerceSetRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_commerce.Set](db, logger),
	}
}

func (commerceSetRepository *CommerceSetRepository) ListByCatalogId(ctx context.Context, catalogId int32) ([]dao_commerce.Set, error) {
	var sets []dao_commerce.Set
	result := commerceSetRepository.db.WithContext(ctx).
		Model(&dao_commerce.Set{}).
		Where("catalog_id = ?", catalogId).
		Order("id").
		Find(&sets)
	if result.Error != nil {
		commerceSetRepository.logger.ErrorFunction(result.Error, catalogId)
		return nil, result.Error
	}
	return sets, nil
}
