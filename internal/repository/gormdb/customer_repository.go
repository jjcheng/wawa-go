package gormdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"

	"gorm.io/gorm"
)

type CustomerRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_customer.Customer]
}

func NewCustomerRepository(db *gorm.DB, logger *service.Logger) repository.CustomerRepository {
	return &CustomerRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_customer.Customer](db, logger),
	}
}

func (customerRepository *CustomerRepository) GetById(ctx context.Context, id int32) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	if err := customerRepository.db.WithContext(ctx).First(&customer, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) Insert(ctx context.Context, customer *dao_customer.Customer) error {
	original := customerSensitiveSnapshotOf(customer)
	if err := customerRepository.encryptSensitiveFields(customer); err != nil {
		return err
	}
	defer restoreCustomerSensitiveValues(customer, original)
	return customerRepository.Repository.Insert(ctx, customer)
}

func (customerRepository *CustomerRepository) InsertBulk(ctx context.Context, customers []dao_customer.Customer) error {
	originals := make([]customerSensitiveSnapshot, len(customers))
	for i := range customers {
		originals[i] = customerSensitiveSnapshotOf(&customers[i])
		if err := customerRepository.encryptSensitiveFields(&customers[i]); err != nil {
			return err
		}
	}
	defer func() {
		for i := range customers {
			restoreCustomerSensitiveValues(&customers[i], originals[i])
		}
	}()
	return customerRepository.Repository.InsertBulk(ctx, customers)
}

func (customerRepository *CustomerRepository) Update(ctx context.Context, customer *dao_customer.Customer) error {
	original := customerSensitiveSnapshotOf(customer)
	if err := customerRepository.encryptSensitiveFields(customer); err != nil {
		return err
	}
	defer restoreCustomerSensitiveValues(customer, original)
	return customerRepository.Repository.Update(ctx, customer)
}

func (customerRepository *CustomerRepository) GetByIdAndUserId(ctx context.Context, id int32, userId int32) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("id = ? AND user_id = ?", id, userId).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, id, userId)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) CountActiveByUserId(ctx context.Context, userId int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND status = ?", userId, types.CustomerStatusActive).
		Count(&count).Error; err != nil {
		customerRepository.logger.ErrorFunction(err, userId)
		return 0, err
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) CountActiveByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Table("customer.customers").
		Joins("JOIN wa.phone_numbers ON wa.phone_numbers.user_id = customer.customers.user_id").
		Where("wa.phone_numbers.business_account_id = ? AND customer.customers.status = ?", businessAccountId, types.CustomerStatusActive).
		Distinct("customer.customers.id").
		Count(&count).Error; err != nil {
		customerRepository.logger.ErrorFunction(err, businessAccountId)
		return 0, err
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByCountryCodePhoneNumber(ctx context.Context, userId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	phoneNumberHash, err := hashStoredSecret(phoneNumber)
	if err != nil {
		return nil, err
	}
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND country_code = ? AND phone_number_hash = ?", userId, countryCode, phoneNumberHash).First(&customer)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND country_code = ? AND phone_number = ?", userId, countryCode, phoneNumber).First(&customer)
	}
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, countryCode, phoneNumber)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAId(ctx context.Context, userId int32, waId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	waIdHash, err := hashStoredSecret(waId)
	if err != nil {
		return nil, err
	}
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND wa_id_hash = ?", userId, waIdHash).
		First(&customer)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND wa_id = ?", userId, waId).First(&customer)
	}
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, waId)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetByMetaUserId(ctx context.Context, userId int32, bsuid string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND meta_user_id = ?", userId, bsuid).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, bsuid)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAIdOrMetaUserId(ctx context.Context, userId int32, waId string, metaUserId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	query := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ?", userId)
	if waId != "" && metaUserId != "" {
		waIdHash, err := hashStoredSecret(waId)
		if err != nil {
			return nil, err
		}
		query = query.Where("wa_id_hash = ? OR meta_user_id = ?", waIdHash, metaUserId)
	} else if waId != "" {
		waIdHash, err := hashStoredSecret(waId)
		if err != nil {
			return nil, err
		}
		query = query.Where("wa_id_hash = ?", waIdHash)
	} else {
		query = query.Where("meta_user_id = ?", metaUserId)
	}
	result := query.First(&customer)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		legacyQuery := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ?", userId)
		if waId != "" && metaUserId != "" {
			legacyQuery = legacyQuery.Where("wa_id = ? OR meta_user_id = ?", waId, metaUserId)
		} else if waId != "" {
			legacyQuery = legacyQuery.Where("wa_id = ?", waId)
		} else {
			legacyQuery = legacyQuery.Where("meta_user_id = ?", metaUserId)
		}
		result = legacyQuery.First(&customer)
	}
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, waId, metaUserId)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) ListByIds(ctx context.Context, userId int32, ids []int32) ([]dao_customer.Customer, error) {
	if len(ids) == 0 {
		return []dao_customer.Customer{}, nil
	}
	var customers []dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND id IN ?", userId, ids).
		Order("id").
		Find(&customers)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, ids)
		return nil, result.Error
	}
	for i := range customers {
		if err := customerRepository.decryptSensitiveFields(&customers[i]); err != nil {
			return nil, err
		}
	}
	return customers, nil
}

func (customerRepository *CustomerRepository) CountByIds(ctx context.Context, userId int32, ids []int32) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	var count int64
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND id IN ?", userId, ids).
		Count(&count)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, ids)
		return 0, result.Error
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByImportedPhoneNumber(ctx context.Context, userId int32, importedPhoneNumber string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	importedPhoneNumberHash, err := hashStoredSecret(importedPhoneNumber)
	if err != nil {
		return nil, err
	}
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND imported_phone_number_hash = ?", userId, importedPhoneNumberHash).
		First(&customer)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		result = customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND imported_phone_number = ?", userId, importedPhoneNumber).First(&customer)
	}
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, userId, importedPhoneNumber)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetByToken(ctx context.Context, token string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("LOWER(token) = LOWER(?)", token).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			customerRepository.logger.ErrorFunction(result.Error, token)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, err
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetDistinctTags(ctx context.Context, userId int32) ([]string, error) {
	var tags []string
	result := customerRepository.db.WithContext(ctx).
		Raw("SELECT DISTINCT unnest(tags) FROM customer.customers WHERE user_id = ? ORDER BY 1", userId).
		Scan(&tags)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId)
		return nil, result.Error
	}
	return tags, nil
}

func (customerRepository *CustomerRepository) List(ctx context.Context, userId int32, name string, phoneNumber string, order types.OrderCustomersType, status types.CustomerStatus, tags []string, page int, pageSize int) (customers []dao_customer.Customer, totalItems int, totalPages int, err error) {
	query := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ?", userId)
	if name != "" {
		query = query.Where("display_name ILIKE ?", "%"+name+"%")
	}
	// Phone-number filtering happens after decryption because the database stores only ciphertext.
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if len(tags) > 0 {
		query = query.Where("tags && ?", pq.Array(tags))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		customerRepository.logger.ErrorFunction(err, userId, name, phoneNumber, order, status, tags)
		return nil, 0, 0, err
	}
	if order == types.OrderCustomersTypeFromOld {
		query = query.Order("id")
	} else {
		query = query.Order("id DESC")
	}
	result := query.Find(&customers)
	if result.Error != nil {
		customerRepository.logger.ErrorFunction(result.Error, userId, name, phoneNumber, order, status, tags, page, pageSize)
		return nil, 0, 0, result.Error
	}
	decrypted := customers[:0]
	for i := range customers {
		if err := customerRepository.decryptSensitiveFields(&customers[i]); err != nil {
			return nil, 0, 0, err
		}
		if phoneNumber == "" || strings.Contains(strings.ToLower(customers[i].PhoneNumber), strings.ToLower(phoneNumber)) {
			decrypted = append(decrypted, customers[i])
		}
	}
	totalItems = len(decrypted)
	totalPages = (totalItems + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	if start >= totalItems {
		return []dao_customer.Customer{}, totalItems, totalPages, nil
	}
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}
	return decrypted[start:end], totalItems, totalPages, nil
}

type customerSensitiveSnapshot struct {
	phoneNumber         string
	waId                string
	additionalData      map[string]any
	importedPhoneNumber string
}

func customerSensitiveSnapshotOf(customer *dao_customer.Customer) customerSensitiveSnapshot {
	return customerSensitiveSnapshot{customer.PhoneNumber, customer.WAId, customer.AdditionalData, customer.ImportedPhoneNumber}
}

func restoreCustomerSensitiveValues(customer *dao_customer.Customer, values customerSensitiveSnapshot) {
	customer.PhoneNumber = values.phoneNumber
	customer.WAId = values.waId
	customer.AdditionalData = values.additionalData
	customer.ImportedPhoneNumber = values.importedPhoneNumber
}

func (customerRepository *CustomerRepository) encryptSensitiveFields(customer *dao_customer.Customer) error {
	if customer.PhoneNumber != "" {
		hash, err := hashStoredSecret(customer.PhoneNumber)
		if err != nil {
			return err
		}
		customer.PhoneNumberHash = hash
		encrypted, err := encryptStoredSecret(customer.PhoneNumber, customerSecretAAD(customer, "phone_number"))
		if err != nil {
			return err
		}
		customer.PhoneNumberEncrypted = encrypted
		customer.PhoneNumber = ""
	}
	if customer.WAId != "" {
		hash, err := hashStoredSecret(customer.WAId)
		if err != nil {
			return err
		}
		customer.WAIdHash = hash
		encrypted, err := encryptStoredSecret(customer.WAId, customerSecretAAD(customer, "wa_id"))
		if err != nil {
			return err
		}
		customer.WAIdEncrypted = encrypted
		customer.WAId = ""
	}
	if customer.ImportedPhoneNumber != "" {
		hash, err := hashStoredSecret(customer.ImportedPhoneNumber)
		if err != nil {
			return err
		}
		customer.ImportedPhoneNumberHash = hash
		encrypted, err := encryptStoredSecret(customer.ImportedPhoneNumber, customerSecretAAD(customer, "imported_phone_number"))
		if err != nil {
			return err
		}
		customer.ImportedPhoneNumberEncrypted = encrypted
		customer.ImportedPhoneNumber = ""
	}
	if customer.AdditionalData != nil {
		payload, err := json.Marshal(customer.AdditionalData)
		if err != nil {
			return err
		}
		encrypted, err := encryptStoredSecret(string(payload), customerSecretAAD(customer, "additional_data"))
		if err != nil {
			return err
		}
		customer.AdditionalDataEncrypted = encrypted
		customer.AdditionalData = nil
	}
	return nil
}

func (customerRepository *CustomerRepository) decryptSensitiveFields(customer *dao_customer.Customer) error {
	if customer.PhoneNumberEncrypted != "" {
		value, err := decryptStoredSecret(customer.PhoneNumberEncrypted, customerSecretAAD(customer, "phone_number"))
		if err != nil {
			return err
		}
		customer.PhoneNumber = value
	}
	if customer.WAIdEncrypted != "" {
		value, err := decryptStoredSecret(customer.WAIdEncrypted, customerSecretAAD(customer, "wa_id"))
		if err != nil {
			return err
		}
		customer.WAId = value
	}
	if customer.ImportedPhoneNumberEncrypted != "" {
		value, err := decryptStoredSecret(customer.ImportedPhoneNumberEncrypted, customerSecretAAD(customer, "imported_phone_number"))
		if err != nil {
			return err
		}
		customer.ImportedPhoneNumber = value
	}
	if customer.AdditionalDataEncrypted != "" {
		value, err := decryptStoredSecret(customer.AdditionalDataEncrypted, customerSecretAAD(customer, "additional_data"))
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(value), &customer.AdditionalData); err != nil {
			return err
		}
	}
	return nil
}

func customerSecretAAD(customer *dao_customer.Customer, purpose string) string {
	identity := customer.Token
	if identity == "" {
		identity = fmt.Sprintf("id:%d", customer.Id)
	}
	return "customer.customers:" + purpose + ":" + identity
}
