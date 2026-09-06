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

func (messageRepository *WAMessageRepository) List(ctx context.Context, phoneNumberId string, customerMetaId string, customerPhoneNumber string) ([]dao_wa.Message, error) {
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
	var messages []dao_wa.Message
	if err := query.Order("timestamp DESC, id DESC").Find(&messages).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerMetaId, customerPhoneNumber)
		return nil, err
	}
	return messages, nil
}
