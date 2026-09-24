package gormdb

import (
	"context"
	"errors"
	"fmt"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type AccountUserPhoneNumberRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.UserPhoneNumber]
}

func NewAccountUserPhoneNumberRepository(db *gorm.DB, logger *service.Logger) repository.AccountUserPhoneNumberRepository {
	return &AccountUserPhoneNumberRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.UserPhoneNumber](db, logger),
	}
}

func (accountUserPhoneNumberRepository *AccountUserPhoneNumberRepository) ListByUserId(ctx context.Context, userId int32) ([]dao_account.UserPhoneNumber, error) {
	var userPhoneNumbers []dao_account.UserPhoneNumber
	if err := accountUserPhoneNumberRepository.db.WithContext(ctx).
		Preload("User").
		Preload("PhoneNumber").
		Where("user_id = ?", userId).
		Order("id").
		Find(&userPhoneNumbers).Error; err != nil {
		return nil, fmt.Errorf("AccountUserPhoneNumberRepository.ListByUserId userId=%d error=%w", userId, err)
	}
	return userPhoneNumbers, nil
}

func (accountUserPhoneNumberRepository *AccountUserPhoneNumberRepository) ListByPhoneNumberId(ctx context.Context, phoneNumberId int32) ([]dao_account.UserPhoneNumber, error) {
	var userPhoneNumbers []dao_account.UserPhoneNumber
	if err := accountUserPhoneNumberRepository.db.WithContext(ctx).
		Preload("User").
		Preload("PhoneNumber").
		Where("phone_number_id = ?", phoneNumberId).
		Order("id").
		Find(&userPhoneNumbers).Error; err != nil {
		return nil, fmt.Errorf("AccountUserPhoneNumberRepository.ListByPhoneNumberId phoneNumberId=%d error=%w", phoneNumberId, err)
	}
	return userPhoneNumbers, nil
}

func (accountUserPhoneNumberRepository *AccountUserPhoneNumberRepository) CountByBusinessAccountIdAndPhoneNumberIds(ctx context.Context, businessAccountId int32, phoneNumberIds []int32) (int, error) {
	if len(phoneNumberIds) == 0 {
		return 0, nil
	}
	var count int64
	result := accountUserPhoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers").
		Where("business_account_id = ? AND id IN ?", businessAccountId, phoneNumberIds).
		Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("AccountUserPhoneNumberRepository.CountByBusinessAccountIdAndPhoneNumberIds businessAccountId=%d phoneNumberIds=%v error=%w", businessAccountId, phoneNumberIds, result.Error)
	}
	return int(count), nil
}

func (accountUserPhoneNumberRepository *AccountUserPhoneNumberRepository) DeleteByUserId(ctx context.Context, userId int32) error {
	if err := accountUserPhoneNumberRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Delete(&dao_account.UserPhoneNumber{}).Error; err != nil {
		return fmt.Errorf("AccountUserPhoneNumberRepository.DeleteByUserId userId=%d error=%w", userId, err)
	}
	return nil
}

func (accountUserPhoneNumberRepository *AccountUserPhoneNumberRepository) GetByUserIdAndPhoneNumberId(ctx context.Context, userId int32, phoneNumberId int32) (*dao_account.UserPhoneNumber, error) {
	var userPhoneNumber dao_account.UserPhoneNumber
	result := accountUserPhoneNumberRepository.db.WithContext(ctx).
		Preload("User").
		Preload("PhoneNumber").
		Where("user_id = ? AND phone_number_id = ?", userId, phoneNumberId).
		First(&userPhoneNumber)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}
		return nil, fmt.Errorf("AccountUserPhoneNumberRepository.GetByUserIdAndPhoneNumberId userId=%d phoneNumberId=%d error=%w", userId, phoneNumberId, result.Error)
	}
	return &userPhoneNumber, nil
}
