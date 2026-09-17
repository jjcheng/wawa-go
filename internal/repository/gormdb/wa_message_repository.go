package gormdb

import (
	"context"
	"strings"
	"time"

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

func (messageRepository *WAMessageRepository) List(ctx context.Context, phoneNumberId int32, customerId int32, ignoreUnsupportedType bool, page int, pageSize int) (messages []dao_wa.Message, totalPages int, totalCount int, err error) {
	query := messageRepository.db.WithContext(ctx).Model(&dao_wa.Message{})
	query = query.Where("phone_number_id = ? AND customer_id = ?", phoneNumberId, customerId)
	if ignoreUnsupportedType {
		query = query.Where("type <> ?", "unsupported")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerId)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	offset := (page - 1) * pageSize
	if err = query.Order("timestamp").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, phoneNumberId, customerId, page, pageSize)
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

func (messageRepository *WAMessageRepository) GetByToken(ctx context.Context, token string) (*dao_wa.Message, error) {
	var message dao_wa.Message
	if err := messageRepository.db.WithContext(ctx).
		Where("token = ?", strings.TrimSpace(token)).
		First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

func (messageRepository *WAMessageRepository) ListNeedResend(ctx context.Context) ([]dao_wa.Message, error) {
	var messages []dao_wa.Message
	result := messageRepository.db.WithContext(ctx).
		Model(&dao_wa.Message{}).
		Where("status = ? AND next_attempt_at IS NOT NULL AND next_attempt_at <= NOW()", "REJECTED").
		Order("next_attempt_at, id").
		Find(&messages)
	if result.Error != nil {
		messageRepository.logger.ErrorFunction(result.Error)
		return nil, result.Error
	}
	return messages, nil
}

func (messageRepository *WAMessageRepository) UpdateNextAttemptAt(ctx context.Context, id int32, nextAttemptAt time.Time) (bool, error) {
	result := messageRepository.db.WithContext(ctx).
		Model(&dao_wa.Message{}).
		Where("id = ? AND status = ? AND next_attempt_at IS NOT NULL AND next_attempt_at <= NOW()", id, types.WAMessageStatusRejected).
		Updates(map[string]any{
			"next_attempt_at": nextAttemptAt,
		})
	if result.Error != nil {
		messageRepository.logger.ErrorFunction(result.Error, id)
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}
