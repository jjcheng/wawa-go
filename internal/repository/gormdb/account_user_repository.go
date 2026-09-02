package gormdb

import (
	"context"
	"errors"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"gorm.io/gorm"
)

type AccountUserRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.User]
}

func NewAccountUserRepository(db *gorm.DB, logger *service.Logger) repository.AccountUserRepository {
	accountUserRepository := AccountUserRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.User](db, logger),
	}
	return &accountUserRepository
}

func (accountUserRepository *AccountUserRepository) Get(ctx context.Context, id int32) (*dao_account.User, error) {
	var item *dao_account.User
	result := accountUserRepository.db.Model(&dao_account.User{}).Where("id = ?", id).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			accountUserRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	return item, nil
}

func (accountUserRepository *AccountUserRepository) HasMasterUser(ctx context.Context, metaBusinessPortfolioId string) (bool, error) {
	var count int64
	result := accountUserRepository.db.WithContext(ctx).
		Table("account.users").
		Joins("JOIN wa.user_phone_numbers ON wa.user_phone_numbers.user_id = account.users.id").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.id = wa.user_phone_numbers.phone_number_id").
		Where("wa.phone_numbers.meta_business_portfolio_id = ? AND account.users.type = ?", metaBusinessPortfolioId, types.UserTypeMaster).
		Count(&count)
	if result.Error != nil {
		accountUserRepository.logger.ErrorFunction(result.Error, metaBusinessPortfolioId)
		return false, result.Error
	}
	return count > 0, nil
}

func (accountUserRepository *AccountUserRepository) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*dao_account.User, error) {
	var item *dao_account.User
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("phone_number = ?", phoneNumber).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			accountUserRepository.logger.ErrorFunction(result.Error, phoneNumber)
		}
		return nil, result.Error
	}
	return item, nil
}

func (accountUserRepository *AccountUserRepository) GetByAccessTokenHash(ctx context.Context, accessTokenHash string) (*dao_account.User, error) {
	var item *dao_account.User
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("access_token_hash = ?", accessTokenHash).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			accountUserRepository.logger.ErrorFunction(result.Error, accessTokenHash)
		}
		return nil, result.Error
	}
	return item, nil
}

func (accountUserRepository *AccountUserRepository) GetByEmailOrPhoneNumber(ctx context.Context, email string, phoneNumber string) (*dao_account.User, error) {
	var item *dao_account.User
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("email = ? OR phone_number = ?", email, phoneNumber).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			accountUserRepository.logger.ErrorFunction(result.Error, email, phoneNumber)
		}
		return nil, result.Error
	}
	return item, nil
}
