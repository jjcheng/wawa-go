package gormdb

import (
	"context"
	"errors"
	"net/http"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"gorm.io/gorm"
)

type CMMessageRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_cm.Message]
}

func NewCMMessageRepository(db *gorm.DB, logger *service.Logger) repository.CMMessageRepository {
	messageRepository := CMMessageRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_cm.Message](db, logger),
	}
	return &messageRepository
}

func (messageRepository *CMMessageRepository) GetByIdentifier(ctx context.Context, identifier string) (*dao_cm.Message, *exception.Exception) {
	var item *dao_cm.Message
	result := messageRepository.db.Model(&dao_cm.Message{}).Where("identifier = ?", identifier).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("message not found", http.StatusNotFound)
		}
		messageRepository.logger.ErrorFunction(result.Error, identifier)
		return nil, exception.NewCustomException("error getting message", http.StatusInternalServerError)
	}
	return item, nil
}

func (messageRepository *CMMessageRepository) ListLastMessagesByConversationIds(ctx context.Context, conversationIds []int32, roles []types.ChatMessageRole) ([]dao_cm.Message, *exception.Exception) {
	var messages []dao_cm.Message
	if err := messageRepository.db.Raw(
		`SELECT DISTINCT ON (conversation_id) id, conversation_id, role, content, date_time FROM cm_message WHERE conversation_id IN (?) AND role in (?) ORDER BY conversation_id, date_time DESC;`,
		conversationIds, roles).Scan(&messages).Error; err != nil {
		messageRepository.logger.ErrorFunction(err, conversationIds, roles)
		return nil, exception.NewCustomException("error getting last messages", http.StatusInternalServerError)
	}
	return messages, nil
}

func (messageRepository *CMMessageRepository) ListByUserIdAndConversationId(ctx context.Context, userId int32, conversationId int32, roles []types.ChatMessageRole) ([]dao_cm.Message, *exception.Exception) {
	var items []dao_cm.Message
	model := messageRepository.db.Model(&dao_cm.Message{}).Select("cm_message.*").Joins("INNER JOIN cm_conversation c ON cm_message.conversation_id = c.id").Where("cm_message.conversation_id = ? AND c.user_id = ?", conversationId, userId)
	if len(roles) > 0 {
		model.Where("cm_message.role IN ?", helper.Map(roles, func(r types.ChatMessageRole) string {
			return string(r)
		}))
	}
	result := model.Order("date_time").Find(&items)
	if result.Error != nil {
		messageRepository.logger.ErrorFunction(result.Error, userId, conversationId)
		return nil, exception.NewCustomException("error getting messages", http.StatusInternalServerError)
	}
	return items, nil
}

func (messageRepository *CMMessageRepository) ListByUserIdAndConversationIdentifier(ctx context.Context, userId int32, conversationIdentifier string, roles []types.ChatMessageRole) ([]dao_cm.Message, *exception.Exception) {
	var items []dao_cm.Message
	model := messageRepository.db.Model(&dao_cm.Message{}).Select("cm_message.*").Joins("INNER JOIN cm_conversation c ON cm_message.conversation_id = c.id").Where("c.user_id = ? AND c.identifier = ?", userId, conversationIdentifier)
	if len(roles) > 0 {
		model.Where("cm_message.role in ?", helper.Map(roles, func(r types.ChatMessageRole) string {
			return string(r)
		}))
	}
	result := model.Order("date_time").Find(&items)
	if result.Error != nil {
		messageRepository.logger.ErrorFunction(result.Error, userId, conversationIdentifier)
		return nil, exception.NewCustomException("error getting messages", http.StatusInternalServerError)
	}
	return items, nil
}

func (messageRepository *CMMessageRepository) GetByConversationIdAndLatestRole(ctx context.Context, conversationId int32, role types.ChatMessageRole) (*dao_cm.Message, *exception.Exception) {
	var item *dao_cm.Message
	result := messageRepository.db.Model(&dao_cm.Message{}).Where("conversation_id = ? AND role = ?", conversationId, role).Order("date_time DESC").First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("message not found", http.StatusNotFound)
		}
		messageRepository.logger.ErrorFunction(result.Error, conversationId, role)
		return nil, exception.NewCustomException("error getting message", http.StatusInternalServerError)
	}
	return item, nil
}

func (messageRepository *CMMessageRepository) GetByConversationIdentifierAndLatestRole(ctx context.Context, userId int32, conversationIdentifier string, role types.ChatMessageRole) (*dao_cm.Message, *exception.Exception) {
	var item *dao_cm.Message
	result := messageRepository.db.Model(&dao_cm.Message{}).
		Select("cm_message.*").
		Joins("INNER JOIN cm_conversation c ON cm_message.conversation_id = c.id").
		Where("c.user_id = ? AND c.identifier = ? AND cm_message.role = ?", userId, conversationIdentifier, role).
		Order("cm_message.date_time DESC").
		First(&item)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("message not found", http.StatusNotFound)
		}
		messageRepository.logger.ErrorFunction(result.Error, userId, conversationIdentifier, role)
		return nil, exception.NewCustomException("error getting message", http.StatusInternalServerError)
	}
	return item, nil
}

func (messageRepository *CMMessageRepository) DeleteBySessionId(ctx context.Context, userId int32, sessionId string) *exception.Exception {
	// Delete messages that match the session_id and belong to the user's conversations
	result := messageRepository.db.Exec(`
		DELETE FROM cm_message
		WHERE session_id = ? AND conversation_id IN (
			SELECT c.id
			FROM cm_conversation c
			WHERE c.user_id = ?
		)`, sessionId, userId)
	if result.Error != nil {
		messageRepository.logger.ErrorFunction(result.Error, userId, sessionId)
		return exception.NewCustomException("error deleting messages by session id", http.StatusInternalServerError)
	}
	return nil
}
