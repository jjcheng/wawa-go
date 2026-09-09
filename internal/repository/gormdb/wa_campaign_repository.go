package gormdb

import (
	"context"
	"math"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type WACampaignRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_customer.Campaign]
}

func NewWACampaignRepository(db *gorm.DB, logger *service.Logger) repository.CampaignRepository {
	return &WACampaignRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_customer.Campaign](db, logger),
	}
}

func (campaignRepository *WACampaignRepository) ListByUserId(ctx context.Context, userId int32, archived bool, status types.CampaignStatus, page int, pageSize int) (*dto.ListResponse[dao_customer.Campaign], error) {
	var campaigns []dao_customer.Campaign
	query := campaignRepository.db.WithContext(ctx).
		Model(&dao_customer.Campaign{}).
		Where("archived = ? AND user_id = ?", archived, userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var numberOfItems int64
	if err := query.Count(&numberOfItems).Error; err != nil {
		campaignRepository.logger.ErrorFunction(err, userId, status, page, pageSize)
		return nil, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&campaigns).Error; err != nil {
		campaignRepository.logger.ErrorFunction(err, userId, status, page, pageSize)
		return nil, err
	}
	numberOfPages := int(math.Ceil(float64(numberOfItems) / float64(pageSize)))
	result := dto.NewPagedListResponse(campaigns, numberOfPages, int(numberOfItems))
	return &result, nil
}
