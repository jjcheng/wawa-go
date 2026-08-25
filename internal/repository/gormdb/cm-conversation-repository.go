package gormdb

import (
	"context"
	"errors"
	"net/http"
	"time"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"gorm.io/gorm"
)

type CMConversationRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_cm.Conversation]
}

func NewCMConversationRepository(db *gorm.DB, logger *service.Logger) repository.CMConversationRepository {
	conversationRepository := CMConversationRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_cm.Conversation](db, logger),
	}
	return &conversationRepository
}

func (conversationRepository *CMConversationRepository) GetByUserIdAndIdentifier(ctx context.Context, userId int32, identifier string) (*dao_cm.Conversation, *exception.Exception) {
	var item *dao_cm.Conversation
	result := conversationRepository.db.Table("cm_conversation").
		Where("user_id = ? AND identifier = ?", userId, identifier).
		First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("conversation not found", http.StatusNotFound)
		}
		conversationRepository.logger.ErrorFunction(result.Error, userId, identifier)
		return nil, exception.NewCustomException("error getting conversation", http.StatusInternalServerError)
	}
	return item, nil
}

func (conversationRepository *CMConversationRepository) GetByUserIdAndId(ctx context.Context, userId int32, id int32) (*dao_cm.Conversation, *exception.Exception) {
	var item *dao_cm.Conversation
	result := conversationRepository.db.Table("cm_conversation").
		Where("user_id = ? AND id = ?", userId, id).
		First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("conversation not found", http.StatusNotFound)
		}
		conversationRepository.logger.ErrorFunction(result.Error, userId, id)
		return nil, exception.NewCustomException("error getting conversation", http.StatusInternalServerError)
	}
	return item, nil
}

func (conversationRepository *CMConversationRepository) CheckConversationExist(ctx context.Context, userId int32, identifier string) (bool, *exception.Exception) {
	var count int64
	result := conversationRepository.db.Model(&dao_cm.Conversation{}).Where("user_id = ? AND identifier = ?", userId, identifier).Count(&count)
	if result.Error != nil {
		conversationRepository.logger.ErrorFunction(result.Error, userId, identifier)
		return false, exception.NewCustomException("error checking conversation exists", http.StatusInternalServerError)
	}
	return count > 0, nil
}

func (conversationRepository *CMConversationRepository) ListByUserId(ctx context.Context, userId int32) ([]dao_cm.Conversation, *exception.Exception) {
	var items []dao_cm.Conversation
	result := conversationRepository.db.Model(&dao_cm.Conversation{}).Where("user_id = ?", userId).Order("id DESC").Find(&items)
	if result.Error != nil {
		conversationRepository.logger.ErrorFunction(result.Error, userId)
		return nil, exception.NewCustomException("error getting conversations", http.StatusInternalServerError)
	}
	return items, nil
}

func (conversationRepository *CMConversationRepository) UpdateTitle(ctx context.Context, id int32, title string) *exception.Exception {
	result := conversationRepository.db.Model(&dao_cm.Conversation{}).Where("id = ?", id).UpdateColumn("title", title)
	if result.Error != nil {
		conversationRepository.logger.ErrorFunction(result.Error, id, title)
		return exception.NewCustomException("error updating conversation title", http.StatusInternalServerError)
	}
	return nil
}

func (conversationRepository *CMConversationRepository) GetStartContext(ctx context.Context, userId int32, identifier string, rateLimitFromDateTime time.Time) (*dao_cm.Conversation, int, *exception.Exception) {
	var result struct {
		dao_cm.Conversation
		MessageCount int `gorm:"column:message_count"`
	}

	// Batch query using JOINs to get conversation, and message count in one query
	err := conversationRepository.db.WithContext(ctx).
		Table("cm_conversation c").
		Select(`
			c.*,
			(SELECT COUNT(*) FROM cm_message m WHERE m.conversation_id = c.id AND m.role = ? AND m.date_time >= ?) as message_count`, string(types.ChatMessageRoleUser), rateLimitFromDateTime).
		Where("c.user_id = ? AND c.identifier = ?", userId, identifier).
		Scan(&result).Error

	if err != nil {
		conversationRepository.logger.ErrorFunction(err, identifier, rateLimitFromDateTime)
		return nil, 0, exception.NewCustomException("error getting conversation context", http.StatusInternalServerError)
	}

	// Check if conversation was found
	if result.Id == 0 {
		return nil, 0, nil
	}

	return &result.Conversation, result.MessageCount, nil
}
