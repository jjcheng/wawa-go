package gormdb

import (
	"context"
	"errors"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
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

func (businessAccountRepository *WABusinessAccountRepository) GetByMetaWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error) {
	var businessAccount *dao_wa.BusinessAccount
	result := businessAccountRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessAccount{}).
		Where("meta_waba_id = ?", metaWABAId).
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		}
		return nil, nil, result.Error
	}
	var businessPortfolio *dao_wa.BusinessPortfolio
	result = businessAccountRepository.db.WithContext(ctx).
		Where("meta_business_portfolio_id = ?", businessAccount.MetaBusinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		}
		return nil, nil, result.Error
	}
	return businessAccount, businessPortfolio, nil
}
