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

func (customerRepository *CustomerRepository) CountActiveByPhoneNumberIds(ctx context.Context, phoneNumberIds []int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("phone_number_id IN ? AND status = ?", phoneNumberIds, types.CustomerStatusActive).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("CustomerRepository.CountActiveByUserId phoneNumberIds=%v error=%w", phoneNumberIds, err)
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) CountByPhoneNumberIds(ctx context.Context, phoneNumberIds []int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("phone_number_id IN ?", phoneNumberIds).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("CustomerRepository.CountByPhoneNumberIds phoneNumberIds=%v error=%w", phoneNumberIds, err)
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) CountActiveByBusinessAccountId(ctx context.Context, businessAccountId int32) (int, error) {
	var count int64
	if err := customerRepository.db.WithContext(ctx).
		Table("customer.customers AS c").
		Joins("JOIN wa.phone_numbers AS pn ON pn.id = c.phone_number_id").
		Where("pn.business_account_id = ? AND c.status = ?", businessAccountId, types.CustomerStatusActive).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("CustomerRepository.CountActiveByBusinessAccountId businessAccountId=%d error=%w", businessAccountId, err)
	}
	return int(count), nil
}

func (customerRepository *CustomerRepository) GetByCountryCodePhoneNumber(ctx context.Context, phoneNumberId int32, countryCode string, phoneNumber string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	phoneNumberHash, err := hashSecret(phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByCountryCodePhoneNumber index=0 phoneNumberId=%d countryCode=%s error=%w", phoneNumberId, countryCode, err)
	}
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("phone_number_id = ? AND country_code = ? and phone_number_hash = ?", phoneNumberId, countryCode, phoneNumberHash).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByCountryCodePhoneNumber index=1 phoneNumberId=%d countryCode=%s error=%w", phoneNumberId, countryCode, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByCountryCodePhoneNumber index=2 phoneNumberId=%d countryCode=%s error=%w", phoneNumberId, countryCode, err)
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByMetaUserId(ctx context.Context, phoneNumberId int32, metaUserId string) (*dao_customer.Customer, error) {
	var customer *dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{}).Where("phone_number_id = ? AND meta_user_id = ?", phoneNumberId, metaUserId).First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByMetaUserId index=0 phoneNumberId=%d metaUserId=%s error=%w", phoneNumberId, metaUserId, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByMetaUserId index=1 phoneNumberId=%d metaUserId=%s error=%w", phoneNumberId, metaUserId, err)
	}
	return customer, nil
}

func (customerRepository *CustomerRepository) GetByWAIdOrMetaUserId(ctx context.Context, phoneNumberId int32, waId string, metaUserId string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	query := customerRepository.db.WithContext(ctx).Model(&dao_customer.Customer{})
	query.Where("phone_number_id = ?", phoneNumberId)
	if waId != "" && metaUserId != "" {
		waIdHash, err := hashSecret(waId)
		if err != nil {
			return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=0 phoneNumberId=%d metaUserId=%s error=%w", phoneNumberId, metaUserId, err)
		}
		query = query.Where("wa_id_hash = ? OR meta_user_id = ?", waIdHash, metaUserId)
	} else if waId != "" {
		waIdHash, err := hashSecret(waId)
		if err != nil {
			return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=1 phoneNumberId=%d metaUserId=%s error=%w", phoneNumberId, metaUserId, err)
		}
		query = query.Where("wa_id_hash = ?", waIdHash)
	} else {
		query = query.Where("meta_user_id = ?", metaUserId)
	}
	result := query.First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=2 phoneNumberId=%d metaUserId=%s error=%w", phoneNumberId, metaUserId, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByWAIdOrMetaUserId index=3 phoneNumberId=%d metaUserId=%s error=%w", phoneNumberId, metaUserId, err)
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) ListByPhoneNumberIdsAndIds(ctx context.Context, phoneNumberIds []int32, ids []int32) ([]dao_customer.Customer, error) {
	if len(ids) == 0 {
		return []dao_customer.Customer{}, nil
	}
	var customers []dao_customer.Customer
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("phone_number_id IN ? AND id IN ?", phoneNumberIds, ids).
		Order("id").
		Find(&customers)
	if result.Error != nil {
		return nil, fmt.Errorf("CustomerRepository.ListByIds index=0 phoneNumberIds=%v ids=%v error=%w", phoneNumberIds, ids, result.Error)
	}
	for i := range customers {
		if err := customerRepository.decryptSensitiveFields(&customers[i]); err != nil {
			return nil, fmt.Errorf("CustomerRepository.ListByIds index=1 phoneNumberIds=%v ids=%v error=%w", phoneNumberIds, ids, result.Error)
		}
	}
	return customers, nil
}

func (customerRepository *CustomerRepository) GetByImportedPhoneNumber(ctx context.Context, phoneNumberId int32, importedPhoneNumber string) (*dao_customer.Customer, error) {
	var customer dao_customer.Customer
	importedPhoneNumberHash, err := hashSecret(importedPhoneNumber)
	if err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByImportedPhoneNumber index=0 phoneNumberId=%d error=%w", phoneNumberId, err)
	}
	result := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("phone_number_id = ? AND imported_phone_number_hash = ?", phoneNumberId, importedPhoneNumberHash).
		First(&customer)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("CustomerRepository.GetByImportedPhoneNumber index=1 phoneNumberId=%d error=%w", phoneNumberId, result.Error)
		}
		return nil, result.Error
	}
	if err := customerRepository.decryptSensitiveFields(&customer); err != nil {
		return nil, fmt.Errorf("CustomerRepository.GetByImportedPhoneNumber index=2 phoneNumberId=%d error=%w", phoneNumberId, err)
	}
	return &customer, nil
}

func (customerRepository *CustomerRepository) GetDistinctTagsByPhoneNumberIds(ctx context.Context, phoneNumberIds []int32) ([]string, error) {
	if len(phoneNumberIds) == 0 {
		return []string{}, nil
	}
	var tags []string
	result := customerRepository.db.WithContext(ctx).
		Raw("SELECT DISTINCT unnest(tags) FROM customer.customers WHERE phone_number_id IN ? ORDER BY 1", phoneNumberIds).
		Scan(&tags)
	if result.Error != nil {
		return nil, fmt.Errorf("CustomerRepository.GetDistinctTags phoneNumberIds=%v error=%w", phoneNumberIds, result.Error)
	}
	return tags, nil
}

func (customerRepository *CustomerRepository) List(ctx context.Context, onlyHasMessage bool, phoneNumberIds []int32, name string, order types.OrderCustomersType, status *types.CustomerStatus, tags []string, page int, pageSize int) (customers []dao_customer.Customer, totalItems int, totalPages int, err error) {
	query := customerRepository.db.WithContext(ctx).
		Table("customer.customers AS c").
		Where("c.phone_number_id IN ?", phoneNumberIds)
	if name != "" {
		query = query.Where("c.display_name ILIKE ?", "%"+name+"%")
	}
	// Phone-number filtering happens after decryption because the database stores only ciphertext.
	if status != nil {
		query = query.Where("c.status = ?", *status)
	}
	if len(tags) > 0 {
		query = query.Where("c.tags && ?", pq.Array(tags))
	}
	if onlyHasMessage {
		query = query.Where(`EXISTS (
			SELECT 1
			FROM wa.messages AS filtered_message
			WHERE filtered_message.customer_id = c.id
				AND filtered_message.phone_number_id = c.phone_number_id
				AND filtered_message.type <> ?
		)`, "unsupported")
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=0 phoneNumberIds=%v name=%s orderBy=%s status=%v tags=%v page=%d pageSize=%d error=%w", phoneNumberIds, name, order, status, tags, page, pageSize, err)
	}
	switch order {
	case types.OrderCustomersTypeLatestMessage:
		query = query.Order("latest_message.timestamp DESC NULLS LAST, latest_message.id DESC NULLS LAST")
	case types.OrderCustomersTypeFromOld:
		query = query.Order("c.id")
	default:
		query = query.Order("c.id DESC")
	}
	query = query.
		Select(`
			c.*,
			latest_message.id AS latest_message_id,
			latest_message.timestamp AS last_message_timestamp,
			latest_message.sending AS latest_message_sending,
			latest_message.type AS latest_message_type,
			latest_message.token AS latest_message_token,
			latest_message.payload_encrypted AS latest_message_payload_encrypted
		`).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT m.id, m.timestamp, m.sending, m.type, m.token, m.payload_encrypted
				FROM wa.messages AS m
				WHERE m.customer_id = c.id
					AND m.phone_number_id = c.phone_number_id
					AND m.type <> ?
				ORDER BY m.timestamp DESC, m.id DESC
				LIMIT 1
			) AS latest_message ON true
		`, "unsupported")
	result := query.Find(&customers)
	if result.Error != nil {
		return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=1 phoneNumberIds=%v name=%s orderBy=%s status=%v tags=%v page=%d pageSize=%d error=%w", phoneNumberIds, name, order, status, tags, page, pageSize, result.Error)
	}
	decrypted := customers[:0]
	for i := range customers {
		if err := customerRepository.decryptSensitiveFields(&customers[i]); err != nil {
			return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=2 phoneNumberIds=%v name=%s orderBy=%s status=%v tags=%v page=%d pageSize=%d error=%w", phoneNumberIds, name, order, status, tags, page, pageSize, err)
		}
		if err := customerRepository.setLatestMessageContent(&customers[i]); err != nil {
			return nil, 0, 0, fmt.Errorf("CustomerRepository.List index=3 phoneNumberIds=%v name=%s orderBy=%s status=%v tags=%v page=%d pageSize=%d error=%w", phoneNumberIds, name, order, status, tags, page, pageSize, err)
		}
		decrypted = append(decrypted, customers[i])
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

func (customerRepository *CustomerRepository) setLatestMessageContent(customer *dao_customer.Customer) error {
	messageType := strings.ToLower(strings.TrimSpace(customer.LatestMessageType))
	if messageType == "" {
		return nil
	}
	content, err := customerRepository.latestMessageContentByType(customer, messageType)
	if err != nil {
		return err
	}
	if content == "" {
		content = titleMessageType(messageType)
	}
	if customer.LatestMessageSending {
		content = "You: " + content
	}
	customer.LatestMessageContent = content
	return nil
}

func (customerRepository *CustomerRepository) latestMessageContentByType(customer *dao_customer.Customer, messageType string) (string, error) {
	switch messageType {
	case "image":
		return "Image", nil
	case "video":
		return "Video", nil
	case "audio":
		return "Audio", nil
	case "document":
		return "Document", nil
	case "text":
		payload, err := customerRepository.latestMessagePayload(customer)
		if err != nil {
			return "", err
		}
		return nestedString(payload, "text", "body"), nil
	case "reaction":
		payload, err := customerRepository.latestMessagePayload(customer)
		if err != nil {
			return "", err
		}
		return nestedString(payload, "reaction", "emoji"), nil
	default:
		return "", nil
	}
}

func (customerRepository *CustomerRepository) latestMessagePayload(customer *dao_customer.Customer) (map[string]any, error) {
	if customer.LatestMessagePayloadEncrypted == "" || customer.LatestMessageToken == "" {
		return nil, nil
	}
	payload, err := decryptSecret(customer.LatestMessagePayloadEncrypted, fmt.Sprintf("wa.messages:%s:%s", "payload", customer.LatestMessageToken))
	if err != nil {
		return nil, fmt.Errorf("CustomerRepository.latestMessagePayload index=0 customerId=%d error=%w", customer.Id, err)
	}
	data := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, fmt.Errorf("CustomerRepository.latestMessagePayload index=1 customerId=%d error=%w", customer.Id, err)
	}
	return data, nil
}

func titleMessageType(messageType string) string {
	if messageType == "" {
		return ""
	}
	return strings.ToUpper(messageType[:1]) + messageType[1:]
}

func nestedString(data map[string]any, key string, nestedKey string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	nested, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	text, _ := nested[nestedKey].(string)
	return text
}

func (customerRepository *CustomerRepository) DeleteByIds(ctx context.Context, ids []int32) error {
	if len(ids) == 0 {
		return nil
	}
	if err := customerRepository.db.WithContext(ctx).
		Where("id IN ?", ids).
		Delete(&dao_customer.Customer{}).Error; err != nil {
		return fmt.Errorf("CustomerRepository.DeleteByIds ids=%v error=%w", ids, err)
	}
	return nil
}

func (customerRepository *CustomerRepository) UpdateStatusByIds(ctx context.Context, ids []int32, status types.CustomerStatus) error {
	if len(ids) == 0 {
		return nil
	}
	if err := customerRepository.db.WithContext(ctx).
		Model(&dao_customer.Customer{}).
		Where("id IN ?", ids).
		Updates(map[string]any{"status": status}).Error; err != nil {
		return fmt.Errorf("CustomerRepository.UpdateStatusByIds ids=%v status=%s error=%w", ids, status, err)
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
