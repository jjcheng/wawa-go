package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type BroadcastRecipientRepository interface {
	Repository[dao_customer.BroadcastRecipient]
	ListByBroadcastId(ctx context.Context, broadcastId int32, name string, status types.WAMessageStatus, onlyMessageCreated bool, page int, pageSize int) (broadcastRecipients []dao_customer.BroadcastRecipient, totalCount int, totalPages int, err error)
	CountMessageStatusesByBroadcastId(ctx context.Context, broadcastId int32) (map[types.WAMessageStatus]int, error)
}
