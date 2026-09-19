package gormdb

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAMessageStatusRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.MessageStatus]
}

func NewWAMessageStatusRepository(db *gorm.DB, logger *service.Logger) repository.WAMessageStatusRepository {
	return &WAMessageStatusRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.MessageStatus](db, logger),
	}
}

func (messageStatusRepository *WAMessageStatusRepository) ListByMessageId(ctx context.Context, messageId int32) ([]dao_wa.MessageStatus, error) {
	var events []dao_wa.MessageStatus
	result := messageStatusRepository.db.WithContext(ctx).
		Model(&dao_wa.MessageStatus{}).
		Where("message_id = ?", messageId).
		Order("timestamp").
		Find(&events)
	if result.Error != nil {
		messageStatusRepository.logger.ErrorFunction(result.Error, messageId)
		return nil, result.Error
	}
	return events, nil
}
