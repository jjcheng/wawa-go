package gormdb

import (
	"context"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAMessageRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.Message]
}

func NewWAMessageRepository(db *gorm.DB, logger *service.Logger) repository.WAMessageRepository {
	return &WAMessageRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.Message](db, logger),
	}
}

func (messageRepository *WAMessageRepository) List(ctx context.Context, phoneNumberId string, customerWAId string, customerMetaUserId string, ignoreUnsupportedType bool, page int, pageSize int) (messages []dao_wa.Message, totalPages int, totalCount int, err error) {
	phoneNumberId = strings.TrimSpace(phoneNumberId)
	customerWAId = strings.TrimSpace(customerWAId)
	customerMetaUserId = strings.TrimSpace(customerMetaUserId)
	query := messageRepository.db.WithContext(ctx).Model(&dao_wa.Message{})
	if phoneNumberId != "" {
		query = query.Where("phone_number_id = ?", phoneNumberId)
	}
	// take either customer_wa_id or customer_meta_user_id
	if customerWAId != "" {
		query = query.Where("customer_wa_id = ?", customerWAId)
	} else if customerMetaUserId != "" {
		query = query.Where("customer_meta_user_id = ?", customerMetaUserId)
	}
	if ignoreUnsupportedType {
		query = query.Where("type <> ?", "unsupported")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerWAId, customerMetaUserId)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize
	if err = query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerWAId, customerMetaUserId, page, pageSize)
		return nil, 0, 0, err
	}
	return messages, totalPages, totalCount, nil
}

func (messageRepository *WAMessageRepository) GetByWAMessageId(ctx context.Context, waMessageId string) (*dao_wa.Message, error) {
	var message dao_wa.Message
	if err := messageRepository.db.WithContext(ctx).
		Where("wa_message_id = ?", strings.TrimSpace(waMessageId)).
		First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}
