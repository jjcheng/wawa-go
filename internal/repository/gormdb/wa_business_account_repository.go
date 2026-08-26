package gormdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WABusinessAccountRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.BusinessAccount]
}

func NewWABusinessAccountRepository(db *gorm.DB, logger *service.Logger) repository.WABusinessAccountRepository {
	return &WABusinessAccountRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.BusinessAccount](db, logger),
	}
}

func (businessAccountRepository *WABusinessAccountRepository) Get(ctx context.Context, id int32) (*dao_wa.BusinessAccount, *exception.Exception) {
	var businessAccount *dao_wa.BusinessAccount
	result := businessAccountRepository.db.WithContext(ctx).Model(&dao_wa.BusinessAccount{}).Where("id = ?", id).First(&businessAccount)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("business account not found", http.StatusNotFound)
		}
		businessAccountRepository.logger.ErrorFunction(result.Error, id)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting business account: %v", result.Error), http.StatusInternalServerError)
	}
	return businessAccount, nil
}

func (businessAccountRepository *WABusinessAccountRepository) GetByMetaWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessAccount, *exception.Exception) {
	var businessAccount *dao_wa.BusinessAccount
	result := businessAccountRepository.db.WithContext(ctx).Model(&dao_wa.BusinessAccount{}).Where("meta_waba_id = ?", metaWABAId).First(&businessAccount)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, exception.NewCustomException("business account not found", http.StatusNotFound)
		}
		businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting business account: %v", result.Error), http.StatusInternalServerError)
	}
	return businessAccount, nil
}

func (businessAccountRepository *WABusinessAccountRepository) CheckExists(ctx context.Context, metaWABAId string) (bool, *exception.Exception) {
	var count int64
	result := businessAccountRepository.db.WithContext(ctx).Model(&dao_wa.BusinessAccount{}).Where("meta_waba_id = ?", metaWABAId).Count(&count)
	if result.Error != nil {
		businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		return false, exception.NewCustomException("error checking business account", http.StatusInternalServerError)
	}
	return count > 0, nil
}

func (businessAccountRepository *WABusinessAccountRepository) ListByMetaWABAIds(ctx context.Context, metaWABAIds []string) ([]dao_wa.BusinessAccount, *exception.Exception) {
	ids := make([]string, 0, len(metaWABAIds))
	for _, id := range metaWABAIds {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []dao_wa.BusinessAccount{}, nil
	}

	var businessAccounts []dao_wa.BusinessAccount
	result := businessAccountRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessAccount{}).
		Where("meta_waba_id IN ?", ids).
		Order("id").
		Find(&businessAccounts)
	if result.Error != nil {
		businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAIds)
		return nil, exception.NewCustomException(fmt.Sprintf("error getting business accounts: %v", result.Error), http.StatusInternalServerError)
	}
	return businessAccounts, nil
}
