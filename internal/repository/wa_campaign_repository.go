package repository

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type WACampaignRepository interface {
	Repository[dao_wa.Campaign]
	ListByUserId(ctx context.Context, userId int32, archived bool, status types.WACampaignStatus, page int, pageSize int) (*dto.ListResponse[dao_wa.Campaign], error)
}
