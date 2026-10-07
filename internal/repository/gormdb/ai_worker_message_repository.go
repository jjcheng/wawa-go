package gormdb

import (
	"context"
	"fmt"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIMessageRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai_worker.Message]
}

func NewAIWorkerMessageRepository(db *gorm.DB, logger *service.Logger) repository.AIWorkerMessageRepository {
	return &AIMessageRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai_worker.Message](db, logger),
	}
}

func (messageRepository *AIMessageRepository) ListByConversationId(ctx context.Context, conversationId int32) ([]dao_ai_worker.Message, error) {
	var messages []dao_ai_worker.Message
	if err := messageRepository.db.WithContext(ctx).
		Where("conversation_id = ?", conversationId).
		Order("id").
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("AIMessageRepository.ListByConversationId conversationId=%d error=%w", conversationId, err)
	}
	return messages, nil
}
