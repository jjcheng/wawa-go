package gormdb

import (
	"context"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
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

func (messageRepository *WAMessageRepository) List(ctx context.Context, phoneNumberId string, customerMetaId string, customerPhoneNumber string, ignoreUnsupportedType bool, page int, pageSize int) (messages []dao_wa.Message, totalPages int, totalCount int, err error) {
	phoneNumberId = strings.TrimSpace(phoneNumberId)
	customerMetaId = strings.TrimSpace(customerMetaId)
	customerPhoneNumber = strings.TrimSpace(customerPhoneNumber)
	query := messageRepository.db.WithContext(ctx).Model(&dao_wa.Message{})
	if phoneNumberId != "" {
		query = query.Where("phone_number_id = ?", phoneNumberId)
	}
	if customerMetaId != "" {
		query = query.Where("customer_meta_user_id = ?", customerMetaId)
	}
	if customerPhoneNumber != "" {
		query = query.Where("customer_phone_number = ?", customerPhoneNumber)
	}
	if ignoreUnsupportedType {
		query = query.Where("type <> ?", "unsupported")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerMetaId, customerPhoneNumber)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize
	if err = query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerMetaId, customerPhoneNumber, page, pageSize)
		return nil, 0, 0, err
	}
	return messages, totalPages, totalCount, nil
}

func (messageRepository *WAMessageRepository) UpdateStatusByWAMessageId(ctx context.Context, metaID string, status types.WAMessageStatus) error {
	result := messageRepository.db.WithContext(ctx).
		Model(&dao_wa.Message{}).
		Where("wa_message_id = ?", strings.TrimSpace(metaID)).
		Update("status", status)
	if result.Error != nil {
		messageRepository.logger.ErrorFunction(result.Error, metaID, status)
		return result.Error
	}
	return nil
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
