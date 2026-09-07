package gormdb

import (
	"context"
	"strings"

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

func (messageStatusEventRepository *WAMessageStatusEventRepository) List(ctx context.Context, waMessageId string) ([]dao_wa.MessageStatusEvent, error) {
	waMessageId = strings.TrimSpace(waMessageId)
	var events []dao_wa.MessageStatusEvent
	if err := messageStatusEventRepository.db.WithContext(ctx).
		Where("wa_message_id = ?", waMessageId).
		Order("timestamp").
		Find(&events).Error; err != nil {
		messageStatusEventRepository.logger.ErrorFunction(err, waMessageId)
		return nil, err
	}
	return events, nil
}

func (messageStatusEventRepository *WAMessageStatusEventRepository) GetLatest(ctx context.Context, waMessageId string) (*dao_wa.MessageStatusEvent, error) {
	waMessageId = strings.TrimSpace(waMessageId)
	var event dao_wa.MessageStatusEvent
	if err := messageStatusEventRepository.db.WithContext(ctx).
		Where("wa_message_id = ?", waMessageId).
		Order("timestamp DESC").
		First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}
