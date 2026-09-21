package repository

import (
	"context"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
)

type CommerceSetRepository interface {
	Repository[dao_commerce.Set]
	ListByCatalogId(ctx context.Context, catalogId int32) ([]dao_commerce.Set, error)
}
