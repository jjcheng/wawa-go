package gormdb

import (
	"context"
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type CampaignRecipientRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_customer.CampaignRecipient]
}

func NewCampaignRecipientRepository(db *gorm.DB, logger *service.Logger) repository.CampaignRecipientRepository {
	return &CampaignRecipientRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_customer.CampaignRecipient](db, logger),
	}
}

func (campaignRecipientRepository *CampaignRecipientRepository) ListByCampaignIdAndUserId(ctx context.Context, campaignID int32, userID int32, name string, status types.CampaignRecipientStatus, page int, pageSize int) (campaignRecipients []dao_customer.CampaignRecipient, totalCount int, totalPages int, err error) {
	query := campaignRecipientRepository.db.WithContext(ctx).
		Model(&dao_customer.CampaignRecipient{}).
		Where("campaign_id = ? AND user_id = ?", campaignID, userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("customer_name ILIKE ?", "%"+name+"%")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		campaignRecipientRepository.logger.ErrorFunction(err, campaignID, userID, status, page, pageSize)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err = query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&campaignRecipients).Error; err != nil {
		campaignRecipientRepository.logger.ErrorFunction(err, campaignID, userID, status, page, pageSize)
		return nil, 0, 0, err
	}
	return campaignRecipients, totalCount, totalPages, nil
}

func (campaignRecipientRepository *CampaignRecipientRepository) CancelByCampaignId(ctx context.Context, campaignID int32) error {
	if err := campaignRecipientRepository.db.WithContext(ctx).
		Model(&dao_customer.CampaignRecipient{}).
		Where("campaign_id = ? AND status = ?", campaignID, types.CampaignRecipientStatusPending).
		Updates(map[string]any{
			"status":      types.CampaignRecipientStatusCancelled,
			"last_update": time.Now(),
		}).Error; err != nil {
		campaignRecipientRepository.logger.ErrorFunction(err, campaignID)
		return err
	}
	return nil
}
