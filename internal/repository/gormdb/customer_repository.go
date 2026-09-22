package gormdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
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
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("id = ?", id).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetById index=0 id=%d error=%w", id, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetById index=1 id=%d error=%w", id, err)
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) Insert(ctx context.Context, customer *dao_customer.Customer) error {
	if customer.Token == "" {
		customer.Token = uuid.NewString()
	}
	original := customerRepository.customerSensitiveSnapshotOf(customer)
	if err := customerRepository.encryptSensitiveFields(customer); err != nil {
		return fmt.Errorf("CustomerRepository.Insert index=0 customerId=%d error=%w", customer.Id, err)
	}
	defer customerRepository.restoreCustomerSensitiveValues(customer, original)
	err := customerRepository.Repository.Insert(ctx, customer)
	if err != nil {
		return fmt.Errorf("CustomerRepository.Insert index=1 customerId=%d error=%w", customer.Id, err)
	}
	return nil
}

func (customerRepository *CustomerRepository) InsertBulk(ctx context.Context, customers []dao_customer.Customer) error {
	originals := make([]customerSensitiveSnapshot, len(customers))
	for i := range customers {
		if customers[i].Token == "" {
			customers[i].Token = uuid.NewString()
		}
		originals[i] = customerRepository.customerSensitiveSnapshotOf(&customers[i])
		if err := customerRepository.encryptSensitiveFields(&customers[i]); err != nil {
			return fmt.Errorf("CustomerRepository.InsertBulk index=0 customers=%d error=%w", len(customers), err)
		}
	}
	defer func() {
		for i := range customers {
			customerRepository.restoreCustomerSensitiveValues(&customers[i], originals[i])
		}
	}()
	err := customerRepository.Repository.InsertBulk(ctx, customers)
	if err != nil {
		return fmt.Errorf("CustomerRepository.InsertBulk index=1 customers=%d error=%w", len(customers), err)
	}
	return nil
}

func (customerRepository *CustomerRepository) Update(ctx context.Context, customer *dao_customer.Customer) error {
	original := customerRepository.customerSensitiveSnapshotOf(customer)
	if err := customerRepository.encryptSensitiveFields(customer); err != nil {
		return fmt.Errorf("CustomerRepository.Update index=0 customerId=%d error=%w", customer.Id, err)
	}
	defer customerRepository.restoreCustomerSensitiveValues(customer, original)
	err := customerRepository.Repository.Update(ctx, customer)
	if err != nil {
		return fmt.Errorf("CustomerRepository.Update index=1 customerId=%d error=%w", customer.Id, err)
	}
	return nil
}

func (customerRepository *CustomerRepository) GetByIdAndUserId(ctx context.Context, id int32, userId int32) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("id = ? AND user_id = ?", id, userId).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByIdAndUserId index=0 id=%d userId=%d error=%w", id, userId, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByIdAndUserId index=1 id=%d userId=%d error=%w", id, userId, result.Error)
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) CountActiveByUserId(ctx context.Context, userId int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND status = ?", userId, types.CustomerStatusActive).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("CustomerRepository.CountActiveByUserId userId=%d error=%w", userId, err)
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
		return 0, fmt.Errorf("CustomerRepository.CountActiveByBusinessAccountId businessAccountId=%d error=%w", businessAccountId, err)
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByCountryCodePhoneNumber(ctx context.Context, userId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	phoneNumberHash, err := hashSecret(phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByCountryCodePhoneNumber index=0 userId=%d countryCode=%s error=%w", userId, countryCode, err)
	}
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND country_code = ? AND phone_number_hash = ?", userId, countryCode, phoneNumberHash).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByCountryCodePhoneNumber index=1 userId=%d countryCode=%s error=%w", userId, countryCode, err)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByCountryCodePhoneNumber index=2 userId=%d countryCode=%s error=%w", userId, countryCode, err)
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAId(ctx context.Context, userId int32, waId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	waIdHash, err := hashSecret(waId)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByWAId index=0 userId=%d error=%w", userId, err)
	}
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND wa_id_hash = ?", userId, waIdHash).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByWAId index=1 userId=%d error=%w", userId, err)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByWAId index=2 userId=%d error=%w", userId, err)
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetByMetaUserId(ctx context.Context, userId int32, bsuid string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ? AND meta_user_id = ?", userId, bsuid).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByMetaUserId index=0 userId=%d bsuid=%s error=%w", userId, bsuid, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByMetaUserId index=1 userId=%d bsuid=%s error=%w", userId, bsuid, result.Error)
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAIdOrMetaUserId(ctx context.Context, userId int32, waId string, metaUserId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	query := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("user_id = ?", userId)
	if waId != "" && metaUserId != "" {
		waIdHash, err := hashSecret(waId)
		if err != nil {
			return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=0 userId=%d metaUserId=%s error=%w", userId, metaUserId, err)
		}
		query = query.Where("wa_id_hash = ? OR meta_user_id = ?", waIdHash, metaUserId)
	} else if waId != "" {
		waIdHash, err := hashSecret(waId)
		if err != nil {
			return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=1 userId=%d metaUserId=%s error=%w", userId, metaUserId, err)
		}
		query = query.Where("wa_id_hash = ?", waIdHash)
	} else {
		query = query.Where("meta_user_id = ?", metaUserId)
	}
	result := query.First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=2 userId=%d metaUserId=%s error=%w", userId, metaUserId, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=3 userId=%d metaUserId=%s error=%w", userId, metaUserId, err)
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
		return nil, fmt.Errorf("CustomerRepository.ListByIds index=0 userId=%d ids=%v error=%w", userId, ids, result.Error)
	}
	for i := range customers {
		if err := customerRepository.decryptSensitiveFields(&customers[i]); err != nil {
			return nil, fmt.Errorf("CustomerRepository.ListByIds index=1 userId=%d ids=%v error=%w", userId, ids, result.Error)
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
		return 0, fmt.Errorf("CustomerRepository.CountByIds userId=%d ids=%v error=%w", userId, ids, result.Error)
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByImportedPhoneNumber(ctx context.Context, userId int32, importedPhoneNumber string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	importedPhoneNumberHash, err := hashSecret(importedPhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByImportedPhoneNumber index=0 userId=%d error=%w", userId, err)
	}
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("user_id = ? AND imported_phone_number_hash = ?", userId, importedPhoneNumberHash).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByImportedPhoneNumber index=1 userId=%d error=%w", userId, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByImportedPhoneNumber index=2 userId=%d error=%w", userId, err)
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
			return nil, fmt.Errorf("CustomerRepository.GetByToken index=0 token=%s error=%w", token, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByToken index=1 token=%s error=%w", token, err)
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetDistinctTags(ctx context.Context, userId int32) ([]string, error) {
	var tags []string
	result := customerRepository.db.WithContext(ctx).
		Raw("SELECT DISTINCT unnest(tags) FROM customer.customers WHERE user_id = ? ORDER BY 1", userId).
		Scan(&tags)
	if result.Error != nil {
		return nil, fmt.Errorf("CustomerRepository.GetDistinctTags userId=%d error=%w", userId, result.Error)
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
		return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=0 userId=%d name=%s orderBy=%s status=%s tags=%v page=%d pageSize=%d error=%w", userId, name, order, status, tags, page, pageSize, err)
	}
	if order == types.OrderCustomersTypeFromOld {
		query = query.Order("id")
	} else {
		query = query.Order("id DESC")
	}
	result := query.Find(&customers)
	if result.Error != nil {
		return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=1 userId=%d name=%s orderBy=%s status=%s tags=%v page=%d pageSize=%d error=%w", userId, name, order, status, tags, page, pageSize, result.Error)
	}
	decrypted := customers[:0]
	for i := range customers {
		if err := customerRepository.decryptSensitiveFields(&customers[i]); err != nil {
			return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=2 userId=%d name=%s orderBy=%s status=%s tags=%v page=%d pageSize=%d error=%w", userId, name, order, status, tags, page, pageSize, err)
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

func (customerRepository *CustomerRepository) DeleteByUserId(ctx context.Context, userId int32) error {
	if err := customerRepository.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Delete(&dao_customer.Customer{}).Error; err != nil {
		return fmt.Errorf("CustomerRepository.DeleteByUserId userId=%d error=%w", userId, err)
	}
	return nil
}

type customerSensitiveSnapshot struct {
	phoneNumber         string
	waId                string
	additionalData      map[string]any
	importedPhoneNumber string
}

func (customerRepository *CustomerRepository) customerSensitiveSnapshotOf(customer *dao_customer.Customer) customerSensitiveSnapshot {
	return customerSensitiveSnapshot{customer.PhoneNumber, customer.WAId, customer.AdditionalData, customer.ImportedPhoneNumber}
}

func (customerRepository *CustomerRepository) restoreCustomerSensitiveValues(customer *dao_customer.Customer, values customerSensitiveSnapshot) {
	customer.PhoneNumber = values.phoneNumber
	customer.WAId = values.waId
	customer.AdditionalData = values.additionalData
	customer.ImportedPhoneNumber = values.importedPhoneNumber
}

func (customerRepository *CustomerRepository) encryptSensitiveFields(customer *dao_customer.Customer) error {
	if customer.PhoneNumber != "" {
		hash, err := hashSecret(customer.PhoneNumber)
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=0 customerId=%d error=%w", customer.Id, err)
		}
		customer.PhoneNumberHash = hash
		encrypted, err := encryptSecret(customer.PhoneNumber, customerRepository.customerSecretAAD(customer, "phone_number"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=1 customerId=%d error=%w", customer.Id, err)
		}
		customer.PhoneNumberEncrypted = encrypted
		customer.PhoneNumber = ""
	}
	if customer.WAId != "" {
		hash, err := hashSecret(customer.WAId)
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=2 customerId=%d error=%w", customer.Id, err)
		}
		customer.WAIdHash = hash
		encrypted, err := encryptSecret(customer.WAId, customerRepository.customerSecretAAD(customer, "wa_id"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=3 customerId=%d error=%w", customer.Id, err)
		}
		customer.WAIdEncrypted = encrypted
		customer.WAId = ""
	}
	if customer.ImportedPhoneNumber != "" {
		hash, err := hashSecret(customer.ImportedPhoneNumber)
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=4 customerId=%d error=%w", customer.Id, err)
		}
		customer.ImportedPhoneNumberHash = hash
		encrypted, err := encryptSecret(customer.ImportedPhoneNumber, customerRepository.customerSecretAAD(customer, "imported_phone_number"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=5 customerId=%d error=%w", customer.Id, err)
		}
		customer.ImportedPhoneNumberEncrypted = encrypted
		customer.ImportedPhoneNumber = ""
	}
	if customer.AdditionalData != nil {
		payload, err := json.Marshal(customer.AdditionalData)
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=6 customerId=%d error=%w", customer.Id, err)
		}
		encrypted, err := encryptSecret(string(payload), customerRepository.customerSecretAAD(customer, "additional_data"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.encryptSensitiveFields index=7 customerId=%d error=%w", customer.Id, err)
		}
		customer.AdditionalDataEncrypted = encrypted
		customer.AdditionalData = nil
	}
	return nil
}

func (customerRepository *CustomerRepository) decryptSensitiveFields(customer *dao_customer.Customer) error {
	if customer.PhoneNumberEncrypted != "" {
		value, err := decryptSecret(customer.PhoneNumberEncrypted, customerRepository.customerSecretAAD(customer, "phone_number"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.decryptSensitiveFields index=0 customerId=%d error=%w", customer.Id, err)
		}
		customer.PhoneNumber = value
	}
	if customer.WAIdEncrypted != "" {
		value, err := decryptSecret(customer.WAIdEncrypted, customerRepository.customerSecretAAD(customer, "wa_id"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.decryptSensitiveFields index=1 customerId=%d error=%w", customer.Id, err)
		}
		customer.WAId = value
	}
	if customer.ImportedPhoneNumberEncrypted != "" {
		value, err := decryptSecret(customer.ImportedPhoneNumberEncrypted, customerRepository.customerSecretAAD(customer, "imported_phone_number"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.decryptSensitiveFields index=2 customerId=%d error=%w", customer.Id, err)
		}
		customer.ImportedPhoneNumber = value
	}
	if customer.AdditionalDataEncrypted != "" {
		value, err := decryptSecret(customer.AdditionalDataEncrypted, customerRepository.customerSecretAAD(customer, "additional_data"))
		if err != nil {
			return fmt.Errorf("CustomerRepository.decryptSensitiveFields index=3 customerId=%d error=%w", customer.Id, err)
		}
		if err := json.Unmarshal([]byte(value), &customer.AdditionalData); err != nil {
			return fmt.Errorf("CustomerRepository.decryptSensitiveFields index=4 customerId=%d error=%w", customer.Id, err)
		}
	}
	return nil
}

func (customerRepository *CustomerRepository) customerSecretAAD(customer *dao_customer.Customer, purpose string) string {
	return fmt.Sprintf("customer.customers:%s:%s", purpose, customer.Token)
}
