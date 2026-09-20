package repository

import (
	"context"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/types"
)

type AccountNotificationRepository interface {
	Repository[dao_account.Notification]
	ListByIds(ctx context.Context, ids []int32, userId int32) ([]dao_account.Notification, error)
	ListByUserId(ctx context.Context, userId int32, typ types.NotificationType, read *bool, page int, pageSize int) (notifications []dao_account.Notification, totalCount int, totalPages int, err error)
	SetStatusByIds(ctx context.Context, userId int32, ids []int32, read bool) error
	DeleteByIds(ctx context.Context, ids []int32, userId int32) error
}
