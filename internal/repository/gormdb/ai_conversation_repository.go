package gormdb

import (
	"context"
	"fmt"

	dao_ai "github.com/jjcheng/wawa-go/internal/dao/ai"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AIConversationRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_ai.Conversation]
}

func NewAIConversationRepository(db *gorm.DB, logger *service.Logger) repository.AIConversationRepository {
	return &AIConversationRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_ai.Conversation](db, logger),
	}
}

func (conversationRepository *AIConversationRepository) ListByUserId(ctx context.Context, userId int32) ([]dao_ai.Conversation, error) {
	var conversations []dao_ai.Conversation
	if err := conversationRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Order("id").
		Find(&conversations).Error; err != nil {
		return nil, fmt.Errorf("AIConversationRepository.ListByUserId userId=%d error=%w", userId, err)
	}
	return conversations, nil
}
