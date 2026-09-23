package repository

import (
	"context"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
)

type CommercePageRepository interface {
	Repository[dao_commerce.Page]
	ListByWebsiteId(ctx context.Context, websiteId int32) ([]dao_commerce.Page, error)
	GetPagesCountByWebsiteId(ctx context.Context, websiteId int32) (int, error)
	ListByOnNavBar(ctx context.Context, websiteId int32) ([]dao_commerce.Page, error)
	GetBySlug(ctx context.Context, websiteId int32, slug string) (*dao_commerce.Page, error)
}
