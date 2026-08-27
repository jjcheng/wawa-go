package gormdb

import (
	"context"
	"errors"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAUserPhoneNumberRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.UserPhoneNumber]
}

func NewWAUserPhoneNumberRepository(db *gorm.DB, logger *service.Logger) repository.WAUserPhoneNumberRepository {
	return &WAUserPhoneNumberRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.UserPhoneNumber](db, logger),
	}
}

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) ListPhoneNumbersByUserId(ctx context.Context, userId int32) ([]dao_wa.PhoneNumber, error) {
	var phoneNumbers []dao_wa.PhoneNumber
	result := userPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers").
		Select("wa.phone_numbers.*").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.phone_number_id = wa.phone_numbers.id").
		Where("wa.user_phone_numbers.user_id = ?", userId).
		Order("wa.phone_numbers.id").
		Find(&phoneNumbers)
	if result.Error != nil {
		userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		return nil, result.Error
	}
	return phoneNumbers, nil
}

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error) {
	// business account
	var businessAccount dao_wa.BusinessAccount
	result := userPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_accounts").
		Select("wa.business_accounts.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.meta_waba_id = wa.business_accounts.meta_waba_id").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.phone_number_id = wa.phone_numbers.id").
		Where("wa.user_phone_numbers.user_id = ?", userId).
		Order("wa.business_accounts.id").
		First(&businessAccount)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("business account not found")
		}
		userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		return nil, nil, result.Error
	}
	// business portfolio
	var businessPortfolio dao_wa.BusinessPortfolio
	result = userPhoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("meta_business_portfolio_id = ?", businessAccount.MetaBusinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("business portfolio not found")
		}
		userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId, businessAccount.MetaBusinessPortfolioId)
		return nil, nil, result.Error
	}
	return &businessPortfolio, &businessAccount, nil
}
