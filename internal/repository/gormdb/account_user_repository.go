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
		return fmt.Errorf("AccountUserReposutory.Insert index=0 userId=%d error=%w", user.Id, err)
	}
	defer accountUserRepository.restoreUserSensitiveFields(user, original)
	err := accountUserRepository.Repository.Insert(ctx, user)
	if err != nil {
		return fmt.Errorf("AccountUserRepository.Insert index=1 userId=%d error=%w", user.Id, err)
	}
	return nil
}

func (accountUserRepository *AccountUserRepository) Update(ctx context.Context, user *dao_account.User) error {
	original := accountUserRepository.userSensitiveSnapshotOf(user)
	if err := accountUserRepository.encryptSensitiveFields(user); err != nil {
		return fmt.Errorf("AccountUserRepository.Update index=0 userId=%d error=%w", user.Id, err)
	}
	defer accountUserRepository.restoreUserSensitiveFields(user, original)
	err := accountUserRepository.Repository.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("AccountUserRepository.Update index=1 userId=%d error=%w", user.Id, err)
	}
	return nil
}

func (accountUserRepository *AccountUserRepository) GetById(ctx context.Context, id int32) (*dao_account.User, error) {
	var item *dao_account.User
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("id = ? AND status <> ?", id, types.UserStatusClosed).First(&item)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("AccountUserRepository.GetById index=0 id=%d error=%w", id, result.Error)
		}
		return nil, result.Error
	}
	if err := accountUserRepository.decryptSensitiveFields(item); err != nil {
		return nil, fmt.Errorf("AccountUserRepository.GetById index=1 id=%d error=%w", id, err)
	}
	if item.Status == types.UserStatusClosed {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

func (accountUserRepository *AccountUserRepository) HasMasterUserInBusinessAccount(ctx context.Context, businessAccountId int32) (bool, error) {
	var count int64
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).
		Where("business_account_id = ? AND type = ? AND status <> ?", businessAccountId, types.UserTypeMaster, types.UserStatusClosed).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("AccountUserRepository.HasMasterUserInBusinessAccount businessAccountId=%d error=%w", businessAccountId, result.Error)
	}
	return count > 0, nil
}

func (accountUserRepository *AccountUserRepository) CountByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error) {
	var count int64
	if err := accountUserRepository.db.WithContext(ctx).
		Model(&dao_account.User{}).
		Where("business_account_id = ?", businessAccountId).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("AccountUserRepository.CountByBusinessAccountId businessAccountId=%d error=%w", businessAccountId, err)
	}
	return int(count), nil
}

func (accountUserRepository *AccountUserRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32, typ *types.UserType) ([]dao_account.User, error) {
	var users []dao_account.User
	query := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("business_account_id = ? AND status <> ?", businessAccountId, types.UserStatusClosed)
	if typ != nil {
		query = query.Where("type = ?", *typ)
	}
	result := query.Order("id").Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("AccountUserRepository.ListByBusinessAccountId index=0 businessAccountId=%d typ=%v error=%w", businessAccountId, typ, result.Error)
	}
	for i := range users {
		if err := accountUserRepository.decryptSensitiveFields(&users[i]); err != nil {
			return nil, fmt.Errorf("AccountUserRepository.ListByBusinessAccountId index=1 businessAccountId=%d typ=%v error=%w", businessAccountId, typ, err)
		}
	}
	return users, nil
}

func (accountUserRepository *AccountUserRepository) GetByIdsAndBusinessAccountId(ctx context.Context, ids []int32, businessAccountId int32) ([]dao_account.User, error) {
	if len(ids) == 0 {
		return []dao_account.User{}, nil
	}
	var users []dao_account.User
	result := accountUserRepository.db.WithContext(ctx).
		Where("id IN ? AND business_account_id = ? AND status <> ?", ids, businessAccountId, types.UserStatusClosed).
		Order("id").
		Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("AccountUserRepository.GetByIdsAndBusinessAccountId ids=%v businessAccountId=%d index=0 error=%w", ids, businessAccountId, result.Error)
	}
	for i := range users {
		if err := accountUserRepository.decryptSensitiveFields(&users[i]); err != nil {
			return nil, fmt.Errorf("AccountUserRepository.GetByIdsAndBusinessAccountId ids=%v businessAccountId=%d index=1 error=%w", ids, businessAccountId, err)
		}
	}
	return users, nil
}

func (accountUserRepository *AccountUserRepository) GetByPhoneNumber(ctx context.Context, countryCode string, phoneNumber string) (*dao_account.User, error) {
	var item *dao_account.User
	phoneNumberHash, err := hashSecret(phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("AccountUserRepository.GetByPhoneNumber index=0 countryCode=%s error=%w", countryCode, err)
	}
	result := accountUserRepository.db.WithContext(ctx).Model(&dao_account.User{}).Where("country_code = ? AND phone_number_hash = ? AND status <> ?", countryCode, phoneNumberHash, types.UserStatusClosed).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}
		return nil, fmt.Errorf("AccountUserRepository.GetByPhoneNumber index=1 countryCode=%s error=%w", countryCode, result.Error)
	}
	if err := accountUserRepository.decryptSensitiveFields(item); err != nil {
		return nil, fmt.Errorf("AccountUserRepository.GetByPhoneNumber index=2 countryCode=%s error=%w", countryCode, err)
	}
	if item.Status == types.UserStatusClosed {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

// encryption
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
			return fmt.Errorf("AccountUserRepository.encryptSensitiveFields index=0 userId=%d error=%w", user.Id, err)
		}
		user.PhoneNumberHash = hash
		encrypted, err := encryptSecret(user.PhoneNumber, accountUserRepository.userSecretAAD(user, "phone_number"))
		if err != nil {
			return fmt.Errorf("AccountUserRepository.encryptSensitiveFields index=1 userId=%d error=%w", user.Id, err)
		}
		user.PhoneNumberEncrypted = encrypted
		user.PhoneNumber = ""
	}
	if user.Email != "" {
		hash, err := hashSecret(user.Email)
		if err != nil {
			return fmt.Errorf("AccountUserRepository.encryptSensitiveFields index=2 userId=%d error=%w", user.Id, err)
		}
		user.EmailHash = hash
		encrypted, err := encryptSecret(user.Email, accountUserRepository.userSecretAAD(user, "email"))
		if err != nil {
			return fmt.Errorf("AccountUserRepository.encryptSensitiveFields index=3 userId=%d error=%w", user.Id, err)
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
			return fmt.Errorf("AccountUserRepository.decryptSensitiveFields index=0 userId=%d error=%w", user.Id, err)
		}
		user.PhoneNumber = phoneNumber
	}
	if user.EmailEncrypted != "" {
		email, err := decryptSecret(user.EmailEncrypted, accountUserRepository.userSecretAAD(user, "email"))
		if err != nil {
			return fmt.Errorf("AccountUserRepository.decryptSensitiveFields index=1 userId=%d error=%w", user.Id, err)
		}
		user.Email = email
	}
	return nil
}
