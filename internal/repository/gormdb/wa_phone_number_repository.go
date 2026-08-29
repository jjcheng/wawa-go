package gormdb

import (
	"context"
	"errors"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAPhoneNumberRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.PhoneNumber]
}

func NewWAPhoneNumberRepository(db *gorm.DB, logger *service.Logger) repository.WAPhoneNumberRepository {
	return &WAPhoneNumberRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.PhoneNumber](db, logger),
	}
}

func (phoneNumberRepository *WAPhoneNumberRepository) Get(ctx context.Context, id int32) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("id = ?", id).First(&phoneNumber)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("phone number not found")
		}
		phoneNumberRepository.logger.ErrorFunction(result.Error, id)
		return nil, result.Error
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetBusinessPortfolioByMetaPhoneNumberId(ctx context.Context, metaPhoneNmberId string) (*dao_wa.BusinessPortfolio, error) {
	var businessPortfolio *dao_wa.BusinessPortfolio
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_portfolios").
		Select("wa.business_portfolios.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.meta_business_portfolio_id = wa.business_portfolios.meta_business_portfolio_id").
		Where("wa.phone_numbers.meta_phone_number_id = ?", metaPhoneNmberId).
		First(&businessPortfolio)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("business portfolio not found")
		}
		phoneNumberRepository.logger.ErrorFunction(result.Error, metaPhoneNmberId)
		return nil, result.Error
	}
	return businessPortfolio, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByBusinessPortfolioId(ctx context.Context, id int32, businessPortfolioId int32) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("id = ? AND business_portfolio_id = ?", id, businessPortfolioId).First(&phoneNumber)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("phone number not found")
		}
		phoneNumberRepository.logger.ErrorFunction(result.Error, id, businessPortfolioId)
		return nil, result.Error
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListByBusinessPortfolioId(ctx context.Context, businessPortfolioId int32) (*[]dao_wa.PhoneNumber, error) {
	var phoneNumbers []dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("business_portfolio_id = ?", businessPortfolioId).Order("id").Find(&phoneNumbers)
	if result.Error != nil {
		phoneNumberRepository.logger.ErrorFunction(result.Error, businessPortfolioId)
		return nil, result.Error
	}
	return &phoneNumbers, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) CheckExists(ctx context.Context, metaPhoneNumberId string) (bool, error) {
	var count int64
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("meta_phone_number_id = ?", metaPhoneNumberId).Count(&count)
	if result.Error != nil {
		phoneNumberRepository.logger.ErrorFunction(result.Error, metaPhoneNumberId)
		return false, result.Error
	}
	return count > 0, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListBusinessAccounts(ctx context.Context, userId int32) ([]dao_wa.BusinessAccount, error) {
	var businessAccounts []dao_wa.BusinessAccount
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_accounts").
		Select("DISTINCT wa.business_accounts.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.meta_waba_id = wa.business_accounts.meta_waba_id").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.phone_number_id = wa.phone_numbers.id").
		Where("wa.user_phone_numbers.user_id = ?", userId).
		Order("wa.business_accounts.id").
		Find(&businessAccounts)
	if result.Error != nil {
		phoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		return nil, result.Error
	}
	return businessAccounts, nil
}
