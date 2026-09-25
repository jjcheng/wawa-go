package gormdb

import (
	"context"
	"errors"
	"fmt"

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

func (phoneNumberRepository *WAPhoneNumberRepository) GetById(ctx context.Context, id int32) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("id = ?", id).First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("PhoneNumberRepository.GetById index=0 id=%d error=%w", id, result.Error)
		}
		return nil, result.Error
	}
	if err := phoneNumberRepository.decryptSecrets(phoneNumber); err != nil {
		return nil, fmt.Errorf("PhoneNumberRepository.GetById index=1 id=%d error=%w", id, result.Error)
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) Insert(ctx context.Context, phoneNumber *dao_wa.PhoneNumber) error {
	displayPhoneNumber := phoneNumber.DisplayPhoneNumber
	waId := phoneNumber.WAId
	registrationPin := phoneNumber.RegistrationPin
	if err := phoneNumberRepository.encryptSecrets(phoneNumber); err != nil {
		return fmt.Errorf("PhoneNumberRepository.Insert index=0 phoneNumberId=%d error=%w", phoneNumber.Id, err)
	}
	defer func() {
		phoneNumber.DisplayPhoneNumber = displayPhoneNumber
		phoneNumber.WAId = waId
		phoneNumber.RegistrationPin = registrationPin
	}()
	err := phoneNumberRepository.Repository.Insert(ctx, phoneNumber)
	if err != nil {
		return fmt.Errorf("PhoneNumberRepository.Insert index=1 phoneNumberId=%d error=%w", phoneNumber.Id, err)
	}
	return nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) Update(ctx context.Context, phoneNumber *dao_wa.PhoneNumber) error {
	displayPhoneNumber := phoneNumber.DisplayPhoneNumber
	waId := phoneNumber.WAId
	registrationPin := phoneNumber.RegistrationPin
	if err := phoneNumberRepository.encryptSecrets(phoneNumber); err != nil {
		return fmt.Errorf("PhoneNumberRepository.Update index=0 phoneNumberId=%d error=%w", phoneNumber.Id, err)
	}
	defer func() {
		phoneNumber.DisplayPhoneNumber = displayPhoneNumber
		phoneNumber.WAId = waId
		phoneNumber.RegistrationPin = registrationPin
	}()
	err := phoneNumberRepository.Repository.Update(ctx, phoneNumber)
	if err != nil {
		return fmt.Errorf("PhoneNumberRepository.Update index=1 phoneNumberId=%d error=%w", phoneNumber.Id, err)
	}
	return nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByMetaPhoneNumberId(ctx context.Context, metaPhoneNumberId string) (*dao_wa.PhoneNumber, error) {
	var phoneNumber *dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).Model(&dao_wa.PhoneNumber{}).Where("meta_phone_number_id = ?", metaPhoneNumberId).First(&phoneNumber)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("PhoneNumberRepository.GetByMetaPhoneNumberId index=0 metaPhoneNumberId=%s error=%w", metaPhoneNumberId, result.Error)
		}
		return nil, result.Error
	}
	if err := phoneNumberRepository.decryptSecrets(phoneNumber); err != nil {
		return nil, fmt.Errorf("PhoneNumberRepository.GetByMetaPhoneNumberId index=1 metaPhoneNumberId=%s error=%w", metaPhoneNumberId, result.Error)
	}
	return phoneNumber, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) CountByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error) {
	var count int64
	if err := phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.PhoneNumber{}).
		Where("business_account_id = ?", businessAccountId).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("PhoneNumberRepository.CountByBusinessAccountId businessAccountId=%d error=%w", businessAccountId, err)
	}
	return int(count), nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListUnassigned(ctx context.Context, businessAccountId int32) ([]dao_wa.PhoneNumber, error) {
	var phoneNumbers []dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers AS pn").
		Where("pn.business_account_id = ?", businessAccountId).
		Where("NOT EXISTS (?)", phoneNumberRepository.db.
			Table("account.user_phone_numbers AS upn").
			Select("1").
			Where("upn.phone_number_id = pn.id")).
		Order("pn.id").
		Find(&phoneNumbers)
	if result.Error != nil {
		return nil, fmt.Errorf("PhoneNumberRepository.ListUnassigned businessAccountId=%d error=%w", businessAccountId, result.Error)
	}
	for i := range phoneNumbers {
		if err := phoneNumberRepository.decryptSecrets(&phoneNumbers[i]); err != nil {
			return nil, fmt.Errorf("PhoneNumberRepository.ListUnassigned businessAccountId=%d phoneNumberId=%d error=%w", businessAccountId, phoneNumbers[i].Id, err)
		}
	}
	return phoneNumbers, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) ListByBusinessAccountId(ctx context.Context, businessAccountId int32, status types.WAPhoneNumberStatus, page int, pageSize int) (phoneNumbers []dao_wa.PhoneNumber, totalCount int, totalPages int, err error) {
	query := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers AS pn").
		Where("pn.business_account_id = ?", businessAccountId)
	if status != "" {
		query = query.Where("pn.status = ?", status)
	} else {
		query = query.Where("pn.status <> ?", types.WAPhoneNumberStatusRemoved)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("PhoneNumberRepository.ListByBusinessAccountId index=0 businessAccountId=%d status=%s page=%d pageSize=%d error=%w", businessAccountId, status, page, pageSize, err)
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err := query.Select("pn.*").Order("pn.id").Offset((page - 1) * pageSize).Limit(pageSize).Find(&phoneNumbers).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("PhoneNumberRepository.ListByBusinessAccountId index=1 businessAccountId=%d status=%s page=%d pageSize=%d error=%w", businessAccountId, status, page, pageSize, err)
	}
	for i := range phoneNumbers {
		if err := phoneNumberRepository.decryptSecrets(&phoneNumbers[i]); err != nil {
			return nil, 0, 0, fmt.Errorf("PhoneNumberRepository.ListByBusinessAccountId index=2 businessAccountId=%d status=%s page=%d pageSize=%d error=%w", businessAccountId, status, page, pageSize, err)
		}
	}
	return phoneNumbers, totalCount, totalPages, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) GetByUserId(ctx context.Context, userId int32) ([]dao_wa.PhoneNumber, *dao_wa.BusinessAccount, *dao_wa.BusinessPortfolio, error) {
	var phoneNumbers []dao_wa.PhoneNumber
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.phone_numbers AS pn").
		Joins("JOIN account.user_phone_numbers AS upn ON upn.phone_number_id = pn.id").
		Where("upn.user_id = ?", userId).
		Select("pn.*").
		Order("pn.id").
		Find(&phoneNumbers)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, nil, fmt.Errorf("PhoneNumberRepository.GetByUserId index=0 userId=%d error=%w", userId, result.Error)
		}
		return nil, nil, nil, result.Error
	}
	// it's ok if there is no phone number
	for i := range phoneNumbers {
		if err := phoneNumberRepository.decryptSecrets(&phoneNumbers[i]); err != nil {
			return nil, nil, nil, fmt.Errorf("PhoneNumberRepository.GetByUserId index=1 userId=%d phoneNumberId=%d error=%w", userId, phoneNumbers[i].Id, err)
		}
	}
	businessPortfolio, businessAccount, err := phoneNumberRepository.GetBusinessPortfolioAndAccountByUserId(ctx, userId)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("PhoneNumberRepository.GetByUserId index=2 userId=%d error=%w", userId, err)
	}
	return phoneNumbers, businessAccount, businessPortfolio, nil
}

// used in update user
func (phoneNumberRepository *WAPhoneNumberRepository) GetBusinessPortfolioAndAccountByUserId(ctx context.Context, userId int32) (*dao_wa.BusinessPortfolio, *dao_wa.BusinessAccount, error) {
	var businessAccount dao_wa.BusinessAccount
	result := phoneNumberRepository.db.WithContext(ctx).
		Table("wa.business_accounts").
		Select("wa.business_accounts.*").
		Joins("JOIN account.users ON account.users.business_account_id = wa.business_accounts.id").
		Where("account.users.id = ?", userId).
		Order("wa.business_accounts.id").
		First(&businessAccount)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("PhoneNumberRepository.GetBusinessPortfolioAndAccountByUserId index=0 userId=%d error=%w", userId, result.Error)
		}
		return nil, nil, result.Error
	}
	var businessPortfolio dao_wa.BusinessPortfolio
	result = phoneNumberRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("id = ?", businessAccount.BussinessPortfolioId).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("PhoneNumberRepository.GetBusinessPortfolioAndAccountByUserId index=1 userId=%d error=%w", userId, result.Error)
		}
		return nil, nil, result.Error
	}
	if err := (phoneNumberRepository).decryptBusinessPortfolioAccessToken(&businessPortfolio); err != nil {
		return nil, nil, fmt.Errorf("PhoneNumberRepository.GetBusinessPortfolioAndAccountByUserId index=2 userId=%d error=%w", userId, result.Error)
	}
	return &businessPortfolio, &businessAccount, nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) decryptSecrets(phoneNumber *dao_wa.PhoneNumber) error {
	if phoneNumber.DisplayPhoneNumberEncrypted != "" {
		displayPhoneNumber, err := decryptSecret(phoneNumber.DisplayPhoneNumberEncrypted, phoneNumberRepository.phoneNumberAAD(phoneNumber, "display_phone_number"))
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.decryptSecrets index=0 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.DisplayPhoneNumber = displayPhoneNumber
	}
	if phoneNumber.WAIdEncrypted != "" {
		waId, err := decryptSecret(phoneNumber.WAIdEncrypted, phoneNumberRepository.phoneNumberAAD(phoneNumber, "wa_id"))
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.decryptSecrets index=1 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.WAId = waId
	}
	if phoneNumber.RegistrationPinEncrypted != "" {
		registrationPin, err := decryptSecret(phoneNumber.RegistrationPinEncrypted, phoneNumberRepository.phoneNumberAAD(phoneNumber, "registration_pin"))
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.decryptSecrets index=2 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.RegistrationPin = registrationPin
	}
	return nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) encryptSecrets(phoneNumber *dao_wa.PhoneNumber) error {
	if phoneNumber.DisplayPhoneNumber != "" {
		encrypted, err := encryptSecret(phoneNumber.DisplayPhoneNumber, phoneNumberRepository.phoneNumberAAD(phoneNumber, "display_phone_number"))
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.encryptSecrets index=0 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.DisplayPhoneNumberEncrypted = encrypted
		phoneNumber.DisplayPhoneNumber = ""
	}
	if phoneNumber.WAId != "" {
		hashed, err := hashSecret(phoneNumber.WAId)
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.encryptSecrets index=1 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.WAIdHash = hashed
		encrypted, err := encryptSecret(phoneNumber.WAId, phoneNumberRepository.phoneNumberAAD(phoneNumber, "wa_id"))
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.encryptSecrets index=2 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.WAIdEncrypted = encrypted
		phoneNumber.WAId = ""
	}
	if phoneNumber.RegistrationPin != "" {
		encrypted, err := encryptSecret(phoneNumber.RegistrationPin, phoneNumberRepository.phoneNumberAAD(phoneNumber, "registration_pin"))
		if err != nil {
			return fmt.Errorf("PhoneNumberRepository.encryptSecrets index=3 phoneNumberId=%d error=%w", phoneNumber.Id, err)
		}
		phoneNumber.RegistrationPinEncrypted = encrypted
		phoneNumber.RegistrationPin = ""
	}
	return nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) decryptBusinessPortfolioAccessToken(businessPortfolio *dao_wa.BusinessPortfolio) error {
	if businessPortfolio.AccessTokenEncrypted == "" {
		return nil
	}
	// make sure the aad is correct here
	accessToken, err := decryptSecret(businessPortfolio.AccessTokenEncrypted, "wa.business_portfolios:access_token:"+businessPortfolio.MetaBusinessPortfolioId)
	if err != nil {
		return fmt.Errorf("PhoneNumberRepository.decryptBusinessPortfolioAccessToken businessPortfolioId=%d error=%w", businessPortfolio.Id, err)
	}
	businessPortfolio.AccessToken = accessToken
	return nil
}

func (phoneNumberRepository *WAPhoneNumberRepository) phoneNumberAAD(phoneNumber *dao_wa.PhoneNumber, purpose string) string {
	return fmt.Sprintf("wa.phone_numbers:%s:%s", purpose, phoneNumber.MetaPhoneNumberId)
}
