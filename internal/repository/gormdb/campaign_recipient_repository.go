package gormdb

import (
	"context"
	"time"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
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

func (campaignRecipientRepository *CampaignRecipientRepository) ListByCampaignId(ctx context.Context, campaignID int32, name string, status types.WAMessageStatus, onlyMessageCreated bool, page int, pageSize int) (campaignRecipients []dao_customer.CampaignRecipient, totalCount int, totalPages int, err error) {
	query := campaignRecipientRepository.db.WithContext(ctx).
		Table("customer.campaign_recipients AS cr").
		Joins("JOIN customer.customers AS c ON c.id = cr.customer_id").
		Joins("LEFT JOIN wa.messages AS m ON m.id = cr.message_id").
		Where("cr.campaign_id = ?", campaignID)
	if status != "" {
		query = query.Where("m.status = ?", status)
	}
	if onlyMessageCreated {
		query = query.Where("m.id IS NOT NULL")
	}
	if name != "" {
		query = query.Where("c.display_name ILIKE ?", "%"+name+"%")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		campaignRecipientRepository.logger.ErrorFunction(err, campaignID, status, page, pageSize)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err = query.Select("cr.*, c.display_name AS customer_name, c.country_code AS customer_country_code, c.phone_number AS customer_phone_number").Order("cr.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&campaignRecipients).Error; err != nil {
		campaignRecipientRepository.logger.ErrorFunction(err, campaignID, status, page, pageSize)
		return nil, 0, 0, err
	}
	messageIds := make([]int32, 0, len(campaignRecipients))
	for _, recipient := range campaignRecipients {
		if recipient.MessageId != nil {
			messageIds = append(messageIds, *recipient.MessageId)
		}
	}
	if len(messageIds) > 0 {
		var messages []dao_wa.Message
		messageQuery := campaignRecipientRepository.db.WithContext(ctx).Where("id IN ?", messageIds)
		if status != "" {
			messageQuery = messageQuery.Where("status = ?", status)
		}
		if err = messageQuery.Order("id DESC").Find(&messages).Error; err != nil {
			campaignRecipientRepository.logger.ErrorFunction(err, campaignID, status, page, pageSize)
			return nil, 0, 0, err
		}
		messageById := make(map[int32]*dao_wa.Message, len(messages))
		for i := range messages {
			messageById[messages[i].Id] = &messages[i]
		}
		for i := range campaignRecipients {
			if campaignRecipients[i].MessageId != nil {
				campaignRecipients[i].Message = messageById[*campaignRecipients[i].MessageId]
			}
		}
	}
	return campaignRecipients, totalCount, totalPages, nil
}

func (campaignRecipientRepository *CampaignRecipientRepository) CountMessageStatusesByCampaignId(ctx context.Context, campaignID int32) (map[types.WAMessageStatus]int, error) {
	type row struct {
		Status types.WAMessageStatus `gorm:"column:status"`
		Count  int                   `gorm:"column:count"`
	}
	rows := []row{}
	if err := campaignRecipientRepository.db.WithContext(ctx).
		Table("customer.campaign_recipients AS cr").
		Joins("JOIN wa.messages AS m ON m.id = cr.message_id").
		Where("cr.campaign_id = ?", campaignID).
		Select("m.status AS status, COUNT(*) AS count").
		Group("m.status").
		Scan(&rows).Error; err != nil {
		campaignRecipientRepository.logger.ErrorFunction(err, campaignID)
		return nil, err
	}
	counts := map[types.WAMessageStatus]int{
		types.WAMessageStatusAccepted:  0,
		types.WAMessageStatusSent:      0,
		types.WAMessageStatusDelivered: 0,
		types.WAMessageStatusRead:      0,
		types.WAMessageStatusFailed:    0,
		types.WAMessageStatusRejected:  0,
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
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
