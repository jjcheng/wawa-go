package gormdb

import (
	"context"
	"net/http"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"gorm.io/gorm"
)

type CMCachedMessageRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_cm.CachedMessage]
}

func NewCMCachedMessageRepository(db *gorm.DB, logger *service.Logger) repository.CMCachedMessageRepository {
	cmCachedMessageRepository := CMCachedMessageRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_cm.CachedMessage](db, logger),
	}
	return &cmCachedMessageRepository
}

func (cmCachedMessageRepository *CMCachedMessageRepository) UpdateReplied(ctx context.Context, identifier string) *exception.Exception {
	if err := cmCachedMessageRepository.db.Model(&dao_cm.CachedMessage{}).Where("identifier = ?", identifier).Update("replied", true).Error; err != nil {
		cmCachedMessageRepository.logger.ErrorFunction(err, identifier)
		return exception.NewCustomException("error updating cached message replied", http.StatusInternalServerError)
	}
	return nil
}

func (cmCachedMessageRepository *CMCachedMessageRepository) CheckHasUnreplied(ctx context.Context, conversationIdentifier string) bool {
	var count int64
	if err := cmCachedMessageRepository.db.Model(&dao_cm.CachedMessage{}).Where("conversation_identifier = ? AND role = ? AND replied = false", conversationIdentifier, string(types.ChatMessageRoleUser)).Count(&count).Error; err != nil {
		cmCachedMessageRepository.logger.ErrorFunction(err, conversationIdentifier)
		return false
	}
	return count > 0
}

func (cmCachedMessageRepository *CMCachedMessageRepository) UpdateAllUnreplied(ctx context.Context, conversationIdentifier string) *exception.Exception {
	if err := cmCachedMessageRepository.db.Model(&dao_cm.CachedMessage{}).Where("conversation_identifier = ?", conversationIdentifier).Update("replied", true).Error; err != nil {
		cmCachedMessageRepository.logger.ErrorFunction(err, conversationIdentifier)
		return exception.NewCustomException("error updating all unreplied cached messages", http.StatusInternalServerError)
	}
	return nil
}
