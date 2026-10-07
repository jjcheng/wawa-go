package gormdb

import (
	"context"
	"fmt"

	dao_ai_worker "github.com/jjcheng/wawa-go/internal/dao/ai_worker"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIConversationRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai_worker.Conversation]
}

func NewAIWorkerConversationRepository(db *gorm.DB, logger *service.Logger) repository.AIWorkerConversationRepository {
	return &AIConversationRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai_worker.Conversation](db, logger),
	}
}

func (conversationRepository *AIConversationRepository) ListByUserId(ctx context.Context, userId int32) ([]dao_ai_worker.Conversation, error) {
	var conversations []dao_ai_worker.Conversation
	if err := conversationRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("id").
		Find(&conversations).Error; err != nil {
		return nil, fmt.Errorf("AIConversationRepository.ListByUserId userId=%d error=%w", userId, err)
	}
	return conversations, nil
}
