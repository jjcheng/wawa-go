package gormdb

import (
	"context"
	"errors"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
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
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByPhoneNumberId(ctx context.Context, phoneNumberId string) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("meta_phone_number_id = ?", phoneNumberId).First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, phoneNumberId)
		}
		return nil, result.Error
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) CountByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string) (int, error) {
	var count int64
	if err := phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.PhoneNumber{}).
		Where("meta_waba_id = ?", metaBusinessAccountId).
		Count(&count).Error; err != nil {
		phoneNumberRepository.logger.ErrorFunction(err, metaBusinessAccountId)
		return 0, err
	}
	return int(count), nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListByMetaBusinessAccountId(ctx context.Context, metaBusinessAccountId string, status types.WAPhoneNumberStatus, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error) {
	query := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers AS pn").
		Joins("INNER JOIN account.users AS u ON u.id = pn.user_id").
		Where("pn.meta_waba_id = ?", metaBusinessAccountId)
	if status != "" {
		query = query.Where("pn.status = ?", status)
	}
	var count int64
	if err := query.Distinct("pn.id").Count(&count).Error; err != nil {
		phoneNumberRepository.logger.ErrorFunction(err, metaBusinessAccountId)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err := query.Select("pn.*, u.name AS user_name").Order("pn.id").Offset((page - 1) * pageSize).Limit(pageSize).Find(&phoneNumbers).Error; err != nil {
		phoneNumberRepository.logger.ErrorFunction(err, metaBusinessAccountId)
		return nil, 0, 0, err
	}
	return phoneNumbers, totalCount, totalPages, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByUserId(ctx context.Context, userId int32) (*dao_wa.PhoneNumber, *dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error) {
	var phoneNumber dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.PhoneNumber{}).
		Where("user_id = ?", userId).
		First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, nil, nil, result.Error
	}
	businessPortfolio, businessAccount, err := phoneNumberRepository.GetBusinessPortfolioAndAccountByUserId(ctx, userId)
	if err != nil {
		return nil, nil, nil, err
	}
	return &phoneNumber, businessAccount, businessPortfolio, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error) {
	var businessAccount dao_wa.BusinessAccount
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_accounts").
		Select("wa.business_accounts.*").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.meta_waba_id = wa.business_accounts.meta_waba_id").
		Where("wa.phone_numbers.user_id = ?", userId).
		Order("wa.business_accounts.id").
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, userId)
		}
		return nil, nil, result.Error
	}
	var businessPortfolio dao_wa.BusinessPortfolio
	result = phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("meta_business_portfolio_id = ?", businessAccount.MetaBusinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			phoneNumberRepository.logger.ErrorFunction(result.Error, userId, businessAccount.MetaBusinessPortfolioId)
		}
		return nil, nil, result.Error
	}
	return &businessPortfolio, &businessAccount, nil
}
