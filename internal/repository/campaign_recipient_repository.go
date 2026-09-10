package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRecipientRepository interface {
	Repository[dao_customer.CampaignRecipient]
	ListByCampaignIdAndUserId(ctx context.Context, campaignId int32, userId int32, name string, status types.CampaignRecipientStatus, page int, pageSize int) (campaignRecipients []dao_customer.CampaignRecipient, totalCount int, totalPages int, err error)
	CancelByCampaignId(ctx context.Context, campaignId int32) error
}
