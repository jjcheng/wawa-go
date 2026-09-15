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

func (campaignRepository *WACampaignRepository) ListByUserId(ctx context.Context, userId int32, name string, status types.CampaignStatus, page int, pageSize int) (*dto.ListResponse[dao_customer.Campaign], error) {
	var campaigns []dao_customer.Campaign
	query := campaignRepository.db.WithContext(ctx).
		Model(&dao_customer.Campaign{}).
		Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
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

func (campaignRepository *WACampaignRepository) CheckNameExist(ctx context.Context, userId int32, name string) (bool, error) {
	var count int64
	err := campaignRepository.db.WithContext(ctx).
		Model(&dao_customer.Campaign{}).
		Where("user_id = ? AND name = ?", userId, name).
		Count(&count).Error
	if err != nil {
		campaignRepository.logger.ErrorFunction(err, userId, name)
		return false, err
	}
	return count > 0, nil
}

func (campaignRepository *WACampaignRepository) ListByIds(ctx context.Context, ids []int32) ([]dao_customer.Campaign, error) {
	if len(ids) == 0 {
		return []dao_customer.Campaign{}, nil
	}
	var campaigns []dao_customer.Campaign
	result := campaignRepository.db.WithContext(ctx).
		Model(&dao_customer.Campaign{}).
		Where("id IN ?", ids).
		Order("id").
		Find(&campaigns)
	if result.Error != nil {
		campaignRepository.logger.ErrorFunction(result.Error, ids)
		return nil, result.Error
	}
	return campaigns, nil
}

func (campaignRepository *WACampaignRepository) ListByMessageIds(ctx context.Context, messageIds []int32) ([]dao_customer.Campaign, error) {
	if len(messageIds) == 0 {
		return []dao_customer.Campaign{}, nil
	}
	var campaigns []dao_customer.Campaign
	result := campaignRepository.db.WithContext(ctx).
		Table("customer.campaigns AS c").
		Joins("JOIN customer.campaign_recipients AS cr ON cr.campaign_id = c.id").
		Where("cr.message_id IN ?", messageIds).
		Distinct("c.*").
		Find(&campaigns)
	if result.Error != nil {
		campaignRepository.logger.ErrorFunction(result.Error, messageIds)
		return nil, result.Error
	}
	return campaigns, nil
}
