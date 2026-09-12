package gormdb

import (
	"context"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAMessageStatusEventRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.MessageStatusEvent]
}

func NewWAMessageStatusEventRepository(db *gorm.DB, logger *service.Logger) repository.WAMessageStatusEventRepository {
	return &WAMessageStatusEventRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.MessageStatusEvent](db, logger),
	}
}

func (messageStatusEventRepository *WAMessageStatusEventRepository) ListByMessageId(ctx context.Context, messageId int32) ([]dao_wa.MessageStatusEvent, error) {
	var events []dao_wa.MessageStatusEvent
	result := messageStatusEventRepository.db.WithContext(ctx).
		Model(&dao_wa.MessageStatusEvent{}).
		Where("message_id = ?", messageId).
		Order("timestamp").
		Find(&events)
	if result.Error != nil {
		messageStatusEventRepository.logger.ErrorFunction(result.Error, messageId)
		return nil, result.Error
	}
	return events, nil
}
