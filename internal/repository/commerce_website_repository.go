package repository

import (
	"context"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
)

type CommerceWebsiteRepository interface {
	Repository[dao_commerce.Website]
	ListByBusinessAccountId(ctx context.Context, businessAccountId int32) ([]dao_commerce.Website, error)
	GetByMetaCatalogId(ctx context.Context, metaCatalogId string) (*dao_commerce.Website, error)
	GetByDomainName(ctx context.Context, domainName string) (*dao_commerce.Website, error)
}
