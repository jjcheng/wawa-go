package gormdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

func (accountUserRepository *AccountUserRepository) Insert(ctx context.Context, user *dao_account.User) error {
	if user.EncryptionID == "" {
		user.EncryptionID = uuid.NewString()
	}
	original := accountUserRepository.userSensitiveSnapshotOf(user)
	if err := accountUserRepository.encryptSensitiveFields(user); err != nil {
		return err
	}
	defer accountUserRepository.restoreUserSensitiveFields(user, original)
	return accountUserRepository.Repository.Insert(ctx, user)
}

func (accountUserRepository *AccountUserRepository) Update(ctx context.Context, user *dao_account.User) error {
	original := accountUserRepository.userSensitiveSnapshotOf(user)
	if err := accountUserRepository.encryptSensitiveFields(user); err != nil {
		return err
	}
	defer accountUserRepository.restoreUserSensitiveFields(user, original)
	return accountUserRepository.Repository.Update(ctx, user)
}

func (accountUserRepository *AccountUserRepository) GetById(ctx context.Context, id int32) (*dao_account.User, error) {
	var item *dao_account.User
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("id = ?", id).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			accountUserRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	if err := accountUserRepository.decryptSensitiveFields(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (accountUserRepository *AccountUserRepository) Get(ctx context.Context, id int32) (*dao_account.User, error) {
	return accountUserRepository.GetById(ctx, id)
}

func (accountUserRepository *AccountUserRepository) HasMasterUser(ctx context.Context, businessAccountId int32) (bool, error) {
	var count int64
	result := accountUserRepository.db.WithContext(ctx).
		Table("account.users").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.user_id = account.users.id").
		Where("wa.phone_numbers.business_account_id = ? AND account.users.type = ?", businessAccountId, types.UserTypeMaster).
		Count(&count)
	if result.Error != nil {
		accountUserRepository.logger.ErrorFunction(result.Error, businessAccountId)
		return false, result.Error
	}
	return count > 0, nil
}

func (accountUserRepository *AccountUserRepository) ListByBusinessPortfolioId(ctx context.Context, businessPortfolioId int32) ([]dao_account.User, error) {
	var users []dao_account.User
	result := accountUserRepository.db.WithContext(ctx).
		Table("account.users").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.user_id = account.users.id").
		Joins("JOIN wa.business_accounts ON wa.business_accounts.id = wa.phone_numbers.business_account_id").
		Where("wa.business_accounts.business_portfolio_id = ?", businessPortfolioId).
		Distinct("account.users.*").
		Order("account.users.id").
		Find(&users)
	if result.Error != nil {
		accountUserRepository.logger.ErrorFunction(result.Error, businessPortfolioId)
		return nil, result.Error
	}
	for i := range users {
		if err := accountUserRepository.decryptSensitiveFields(&users[i]); err != nil {
			return nil, err
		}
	}
	return users, nil
}

func (accountUserRepository *AccountUserRepository) GetByPhoneNumber(ctx context.Context, countryCode string, phoneNumber string) (*dao_account.User, error) {
	var item *dao_account.User
	phoneNumberHash, err := hashSecret(phoneNumber)
	if err != nil {
		return nil, err
	}
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("country_code = ? AND phone_number_hash = ?", countryCode, phoneNumberHash).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			accountUserRepository.logger.ErrorFunction(result.Error, phoneNumber)
		}
		return nil, result.Error
	}
	if err := accountUserRepository.decryptSensitiveFields(item); err != nil {
		return nil, err
	}
	return item, nil
}

type userSensitiveSnapshot struct {
	phoneNumber string
	email       string
}

func (accountUserRepository *AccountUserRepository) userSensitiveSnapshotOf(user *dao_account.User) userSensitiveSnapshot {
	return userSensitiveSnapshot{phoneNumber: user.PhoneNumber, email: user.Email}
}

func (accountUserRepository *AccountUserRepository) restoreUserSensitiveFields(user *dao_account.User, snapshot userSensitiveSnapshot) {
	user.PhoneNumber = snapshot.phoneNumber
	user.Email = snapshot.email
}

func (accountUserRepository *AccountUserRepository) userSecretAAD(user *dao_account.User, purpose string) string {
	return fmt.Sprintf("account.users:%s:%s", purpose, user.EncryptionID)
}

func (accountUserRepository *AccountUserRepository) encryptSensitiveFields(user *dao_account.User) error {
	if user.PhoneNumber != "" {
		hash, err := hashSecret(user.PhoneNumber)
		if err != nil {
			return err
		}
		user.PhoneNumberHash = hash
		encrypted, err := encryptSecret(user.PhoneNumber, accountUserRepository.userSecretAAD(user, "phone_number"))
		if err != nil {
			return err
		}
		user.PhoneNumberEncrypted = encrypted
		user.PhoneNumber = ""
	}
	if user.Email != "" {
		hash, err := hashSecret(user.Email)
		if err != nil {
			return err
		}
		user.EmailHash = hash
		encrypted, err := encryptSecret(user.Email, accountUserRepository.userSecretAAD(user, "email"))
		if err != nil {
			return err
		}
		user.EmailEncrypted = encrypted
		user.Email = ""
	}
	return nil
}

func (accountUserRepository *AccountUserRepository) decryptSensitiveFields(user *dao_account.User) error {
	if user.PhoneNumberEncrypted != "" {
		phoneNumber, err := decryptSecret(user.PhoneNumberEncrypted, accountUserRepository.userSecretAAD(user, "phone_number"))
		if err != nil {
			return err
		}
		user.PhoneNumber = phoneNumber
	}
	if user.EmailEncrypted != "" {
		email, err := decryptSecret(user.EmailEncrypted, accountUserRepository.userSecretAAD(user, "email"))
		if err != nil {
			return err
		}
		user.Email = email
	}
	return nil
}
