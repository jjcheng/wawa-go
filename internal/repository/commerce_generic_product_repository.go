package repository

import (
	"context"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
)

type CommerceGenericProductRepository interface {
	Repository[dao_commerce.GenericProduct]
	List(ctx context.Context, catalogId int32, setId int32, name string, orderBy string, desc bool, page int, pageSize int) (products []dao_commerce.GenericProduct, totalItems int, totalPages int, err error)
}
