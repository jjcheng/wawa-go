package gormdb

import (
	"context"
	"errors"
	"strings"

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

func (businessAccountRepository *WABusinessAccountRepository) Get(ctx context.Context, id int32) (*dao_wa.BusinessAccount, error) {
	var businessAccount *dao_wa.BusinessAccount
	result := businessAccountRepository.db.WithContext(ctx).Model(&dao_wa.BusinessAccount{}).Where("id = ?", id).First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	return businessAccount, nil
}

func (businessAccountRepository *WABusinessAccountRepository) GetByMetaWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessAccount, error) {
	var businessAccount *dao_wa.BusinessAccount
	result := businessAccountRepository.db.WithContext(ctx).Model(&dao_wa.BusinessAccount{}).Where("meta_waba_id = ?", metaWABAId).First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		}
		return nil, result.Error
	}
	return businessAccount, nil
}

func (businessAccountRepository *WABusinessAccountRepository) GetByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessAccount, error) {
	var userPhoneNumber dao_wa.UserPhoneNumber
	var phoneNumber dao_wa.PhoneNumber
	var businessAccount *dao_wa.BusinessAccount

	// Get the user's phone number mapping
	result := businessAccountRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		First(&userPhoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, result.Error
	}

	// Get the phone number details
	result = businessAccountRepository.db.WithContext(ctx).
		Where("id = ?", userPhoneNumber.PhoneNumberId).
		First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, userPhoneNumber.PhoneNumberId)
		}
		return nil, result.Error
	}

	// Get the business account by WABA ID
	result = businessAccountRepository.db.WithContext(ctx).
		Where("meta_waba_id = ?", phoneNumber.MetaWABAId).
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, phoneNumber.MetaWABAId)
		}
		return nil, result.Error
	}

	return businessAccount, nil
}

func (businessAccountRepository *WABusinessAccountRepository) GetBusinessPortfolioByWABAId(ctx context.Context, metaWABAId string) (*dao_wa.BusinessPortfolio, error) {
	var businessPortfolio *dao_wa.BusinessPortfolio
	var businessAccount dao_wa.BusinessAccount

	// First get the business account to find the portfolio ID
	result := businessAccountRepository.db.WithContext(ctx).
		Where("meta_waba_id = ?", metaWABAId).
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		}
		return nil, result.Error
	}

	// Then get the portfolio using the portfolio ID
	result = businessAccountRepository.db.WithContext(ctx).
		Where("meta_business_portfolio_id = ?", businessAccount.MetaBusinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		}
		return nil, result.Error
	}
	return businessPortfolio, nil
}

func (businessAccountRepository *WABusinessAccountRepository) CheckExists(ctx context.Context, metaWABAId string) (bool, error) {
	var count int64
	result := businessAccountRepository.db.WithContext(ctx).Model(&dao_wa.BusinessAccount{}).Where("meta_waba_id = ?", metaWABAId).Count(&count)
	if result.Error != nil {
		businessAccountRepository.logger.ErrorFunction(result.Error, metaWABAId)
		return false, result.Error
	}
	return count > 0, nil
}

func (businessAccountRepository *WABusinessAccountRepository) ListByMetaWABAIds(ctx context.Context, metaWABAIds []string) ([]dao_wa.BusinessAccount, error) {
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
		return nil, result.Error
	}
	return businessAccounts, nil
}

func (businessAccountRepository *WABusinessAccountRepository) GetBusinessAccountAndPortfolioByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error) {
	var userPhoneNumber dao_wa.UserPhoneNumber
	var phoneNumber dao_wa.PhoneNumber
	var businessAccount *dao_wa.BusinessAccount
	var businessPortfolio *dao_wa.BusinessPortfolio

	// Get the user's phone number mapping
	result := businessAccountRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		First(&userPhoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, nil, result.Error
	}

	// Get the phone number details
	result = businessAccountRepository.db.WithContext(ctx).
		Where("id = ?", userPhoneNumber.PhoneNumberId).
		First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, userPhoneNumber.PhoneNumberId)
		}
		return nil, nil, result.Error
	}

	// Get the business account by WABA ID
	result = businessAccountRepository.db.WithContext(ctx).
		Where("meta_waba_id = ?", phoneNumber.MetaWABAId).
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, phoneNumber.MetaWABAId)
		}
		return nil, nil, result.Error
	}

	// Get the business portfolio
	result = businessAccountRepository.db.WithContext(ctx).
		Where("meta_business_portfolio_id = ?", businessAccount.MetaBusinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessAccountRepository.logger.ErrorFunction(result.Error, businessAccount.MetaBusinessPortfolioId)
		}
		return nil, nil, result.Error
	}

	return businessPortfolio, businessAccount, nil
}
