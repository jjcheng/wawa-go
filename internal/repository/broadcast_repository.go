package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type BarodcastRepository interface {
	Repository[dao_customer.Broadcast]
	ListByUserId(ctx context.Context, userId int32, name string, status types.BroadcastStatus, page int, pageSize int) (*dto.ListResponse[dao_customer.Broadcast], error)
	CheckNameExist(ctx context.Context, userId int32, name string) (bool, error)
	ListByIds(ctx context.Context, ids []int32) ([]dao_customer.Broadcast, error)
	ListByMessageIds(ctx context.Context, messageIds []int32) ([]dao_customer.Broadcast, error)
	ListPendingBroadcasts(ctx context.Context) ([]dao_customer.Broadcast, error)
	DeleteByUserId(ctx context.Context, userId int32) error
}
