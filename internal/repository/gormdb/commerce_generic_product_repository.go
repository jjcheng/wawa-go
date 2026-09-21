package gormdb

import (
	"context"
	"fmt"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type CommerceGenericProductRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_commerce.GenericProduct]
}

func NewCommericeGenericProductRepository(db *gorm.DB, logger *service.Logger) repository.CommerceGenericProductRepository {
	return CommerceGenericProductRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_commerce.GenericProduct](db, logger),
	}
}

func (commerceGenericProductRepository CommerceGenericProductRepository) List(ctx context.Context, catalogId int32, setId int32, name string, orderBy string, desc bool, page int, pageSize int) (products []dao_commerce.GenericProduct, totalItems int, totalPages int, err error) {
	query := commerceGenericProductRepository.db.WithContext(ctx).
		Table("commerce.generic_products").
		Joins("JOIN commerce.sets ON commerce.sets.id = commerce.generic_products.set_id").
		Where("commerce.sets.catalog_id = ?", catalogId)
	if setId > 0 {
		query = query.Where("commerce.generic_products.set_id = ?", setId)
	}
	if name != "" {
		query = query.Where("commerce.generic_products.name ILIKE ?", "%"+name+"%")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		commerceGenericProductRepository.logger.ErrorFunction(err, catalogId, setId, name, orderBy, desc, page, pageSize)
		return nil, 0, 0, err
	}
	totalItems = int(count)
	totalPages = (totalItems + pageSize - 1) / pageSize
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	offset := (page - 1) * pageSize
	order := fmt.Sprintf("commerce.generic_products.%s %s", commerceGenericProductRepository.orderColumn(orderBy), direction)
	if err = query.Select("commerce.generic_products.*").Order(order).Offset(offset).Limit(pageSize).Find(&products).Error; err != nil {
		commerceGenericProductRepository.logger.ErrorFunction(err, catalogId, setId, name, orderBy, desc, page, pageSize)
		return nil, 0, 0, err
	}
	return products, totalItems, totalPages, nil
}

// orderColumn whitelists sortable columns so orderBy can never reach raw SQL.
func (CommerceGenericProductRepository) orderColumn(orderBy string) string {
	switch orderBy {
	case "name", "price", "retailer_id":
		return orderBy
	default:
		return "id"
	}
}
