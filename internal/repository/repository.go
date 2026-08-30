package repository

import (
	"context"

	"github.com/jjcheng/wawa-go/internal/dao"
)

type Repository[T dao.DAO] interface {
	// READ
	GetById(ctx context.Context, id int32) (*T, error)
	ListAll(ctx context.Context) ([]T, error)
	// WRITE
	Insert(ctx context.Context, entity *T) error
	InsertBulk(ctx context.Context, entities []T) error // make sure to use unitOfWork.BeginTransaction() to wrap this
	Update(ctx context.Context, entity *T) error
	UpdateFields(ctx context.Context, id int32, fields map[string]any) error
	DeleteById(ctx context.Context, id int32) error
	DeleteAll(ctx context.Context) error
}
