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

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) ListPhoneNumbersByUserId(ctx context.Context, userId int32, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error) {
	query := userPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.phone_number_id = wa.phone_numbers.id").
		Where("wa.user_phone_numbers.user_id = ?", userId)
	var count int64
	if err := query.Distinct("wa.phone_numbers.id").Count(&count).Error; err != nil {
		userPhoneNumberRepository.logger.ErrorFunction(err, userId)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	list := query.Select("wa.phone_numbers.*").Group("wa.phone_numbers.id").Order("wa.phone_numbers.id").Offset((page - 1) * pageSize).Limit(pageSize).Find(&phoneNumbers)
	if list.Error != nil {
		userPhoneNumberRepository.logger.ErrorFunction(list.Error, userId)
		return nil, 0, 0, list.Error
	}
	return phoneNumbers, totalCount, totalPages, nil
}

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) GetByUserIdPhoneNumberId(ctx context.Context, userId int32, phoneNumberId int32) (*dao_wa.UserPhoneNumber, error) {
	var userPhoneNumber *dao_wa.UserPhoneNumber
	result := userPhoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.UserPhoneNumber{}).
		Where("user_id = ? AND phone_number_id = ?", userId, phoneNumberId).
		First(&userPhoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId, phoneNumberId)
		}
		return nil, result.Error
	}
	return userPhoneNumber, nil
}

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) GetByPhoneNumberId(ctx context.Context, phoneNumberId string) (*dao_wa.UserPhoneNumber, error) {
	var userPhoneNumber dao_wa.UserPhoneNumber
	result := userPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.user_phone_numbers").
		Select("wa.user_phone_numbers.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.id = wa.user_phone_numbers.phone_number_id").
		Where("wa.phone_numbers.meta_phone_number_id = ?", phoneNumberId).
		First(&userPhoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userPhoneNumberRepository.logger.ErrorFunction(result.Error, phoneNumberId)
		}
		return nil, result.Error
	}
	return &userPhoneNumber, nil
}

func (userPhoneNumberRepository *WAUserPhoneNumberRepository) GetValidBusinessAccount(ctx context.Context, userId int32, metaWABAId string) (*dao_wa.BusinessAccount, error) {
	var businessAccount dao_wa.BusinessAccount
	result := userPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_accounts").
		Select("wa.business_accounts.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.meta_waba_id = wa.business_accounts.meta_waba_id").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.phone_number_id = wa.phone_numbers.id").
		Where("wa.user_phone_numbers.user_id = ? AND wa.business_accounts.meta_waba_id = ?", userId, metaWABAId).
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId, metaWABAId)
		}
		return nil, result.Error
	}
	return &businessAccount, nil
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
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, nil, result.Error
	}
	// business portfolio
	var businessPortfolio dao_wa.BusinessPortfolio
	result = userPhoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("meta_business_portfolio_id = ?", businessAccount.MetaBusinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userPhoneNumberRepository.logger.ErrorFunction(result.Error, userId, businessAccount.MetaBusinessPortfolioId)
		}
		return nil, nil, result.Error
	}
	return &businessPortfolio, &businessAccount, nil
}
