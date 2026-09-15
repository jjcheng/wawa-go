package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRecipientRepository interface {
	Repository[dao_customer.CampaignRecipient]
	ListByCampaignId(ctx context.Context, campaignId int32, name string, status types.WAMessageStatus, onlyMessageCreated bool, page int, pageSize int) (campaignRecipients []dao_customer.CampaignRecipient, totalCount int, totalPages int, err error)
	CountMessageStatusesByCampaignId(ctx context.Context, campaignId int32) (map[types.WAMessageStatus]int, error)
	CancelByCampaignId(ctx context.Context, campaignId int32) error
}
