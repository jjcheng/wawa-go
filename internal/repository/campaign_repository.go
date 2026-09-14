package repository

import (
	"context"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CampaignRepository interface {
	Repository[dao_customer.Campaign]
	ListByUserId(ctx context.Context, userId int32, name string, status types.CampaignStatus, page int, pageSize int) (*dto.ListResponse[dao_customer.Campaign], error)
	CheckNameExist(ctx context.Context, userId int32, name string) (bool, error)
	ListByIds(ctx context.Context, ids []int32) ([]dao_customer.Campaign, error)
	ListByRecipientIds(ctx context.Context, recipientIds []int32) ([]dao_customer.Campaign, error)
}
