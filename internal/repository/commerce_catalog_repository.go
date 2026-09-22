package repository

import (
	"context"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
)

type CommerceCatalogRepository interface {
	Repository[dao_commerce.Catalog]
	GetByWebsiteId(ctx context.Context, websiteId int32) (*dao_commerce.Catalog, error)
	ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_commerce.Catalog, error)
}
