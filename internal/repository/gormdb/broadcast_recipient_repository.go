package gormdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type BroadcastRecipientRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_customer.BroadcastRecipient]
}

func NewBroadcastRecipientRepository(db *gorm.DB, logger *service.Logger) repository.BroadcastRecipientRepository {
	return &BroadcastRecipientRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_customer.BroadcastRecipient](db, logger),
	}
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) Insert(ctx context.Context, recipient *dao_customer.BroadcastRecipient) error {
	payload := recipient.Payload
	if recipient.EncryptionID == "" {
		recipient.EncryptionID = uuid.NewString()
	}
	if err := broadcastRecipientRepository.encryptPayload(recipient); err != nil {
		return err
	}
	defer func() { recipient.Payload = payload }()
	return broadcastRecipientRepository.Repository.Insert(ctx, recipient)
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) InsertBulk(ctx context.Context, recipients []dao_customer.BroadcastRecipient) error {
	payloads := make([]map[string]any, len(recipients))
	for i := range recipients {
		payloads[i] = recipients[i].Payload
		if recipients[i].EncryptionID == "" {
			recipients[i].EncryptionID = uuid.NewString()
		}
		if err := broadcastRecipientRepository.encryptPayload(&recipients[i]); err != nil {
			return err
		}
	}
	defer func() {
		for i := range recipients {
			recipients[i].Payload = payloads[i]
		}
	}()
	return broadcastRecipientRepository.Repository.InsertBulk(ctx, recipients)
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) Update(ctx context.Context, recipient *dao_customer.BroadcastRecipient) error {
	payload := recipient.Payload
	if err := broadcastRecipientRepository.encryptPayload(recipient); err != nil {
		return err
	}
	defer func() { recipient.Payload = payload }()
	return broadcastRecipientRepository.Repository.Update(ctx, recipient)
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) GetById(ctx context.Context, id int32) (*dao_customer.BroadcastRecipient, error) {
	var recipient dao_customer.BroadcastRecipient
	result := broadcastRecipientRepository.db.WithContext(ctx).
		Model(&dao_customer.BroadcastRecipient{}).
		Where("id = ?", id).
		First(&recipient)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			broadcastRecipientRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	if err := broadcastRecipientRepository.decryptPayload(&recipient); err != nil {
		return nil, err
	}
	return &recipient, nil
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) ListByBroadcastId(ctx context.Context, broadcastId int32, name string, status types.WAMessageStatus, onlyMessageCreated bool, page int, pageSize int) (broadcastRecipients []dao_customer.BroadcastRecipient, totalCount int, totalPages int, err error) {
	query := broadcastRecipientRepository.db.WithContext(ctx).
		Table("customer.broadcast_recipients AS cr").
		Joins("JOIN customer.customers AS c ON c.id = cr.customer_id").
		Joins("LEFT JOIN wa.messages AS m ON m.id = cr.message_id").
		Where("cr.broadcast_id = ?", broadcastId)
	if status != "" {
		query = query.Where("m.status = ?", status)
	}
	if onlyMessageCreated {
		query = query.Where("m.id IS NOT NULL")
	}
	if name != "" {
		query = query.Where("c.display_name ILIKE ?", "%"+name+"%")
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		broadcastRecipientRepository.logger.ErrorFunction(err, broadcastId, status, page, pageSize)
		return nil, 0, 0, err
	}
	totalCount = int(count)
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err = query.Select("cr.*, c.display_name AS customer_name, c.country_code AS customer_country_code, c.phone_number AS customer_phone_number").Order("cr.id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&broadcastRecipients).Error; err != nil {
		broadcastRecipientRepository.logger.ErrorFunction(err, broadcastId, status, page, pageSize)
		return nil, 0, 0, err
	}
	for i := range broadcastRecipients {
		if err = broadcastRecipientRepository.decryptPayload(&broadcastRecipients[i]); err != nil {
			return nil, 0, 0, err
		}
	}
	messageIds := make([]int32, 0, len(broadcastRecipients))
	for _, recipient := range broadcastRecipients {
		if recipient.MessageId != nil {
			messageIds = append(messageIds, *recipient.MessageId)
		}
	}
	if len(messageIds) > 0 {
		var messages []dao_wa.Message
		messageQuery := broadcastRecipientRepository.db.WithContext(ctx).Where("id IN ?", messageIds)
		if status != "" {
			messageQuery = messageQuery.Where("status = ?", status)
		}
		if err = messageQuery.Order("id DESC").Find(&messages).Error; err != nil {
			broadcastRecipientRepository.logger.ErrorFunction(err, broadcastId, status, page, pageSize)
			return nil, 0, 0, err
		}
		messageById := make(map[int32]*dao_wa.Message, len(messages))
		for i := range messages {
			messageById[messages[i].Id] = &messages[i]
		}
		for i := range broadcastRecipients {
			if broadcastRecipients[i].MessageId != nil {
				broadcastRecipients[i].Message = messageById[*broadcastRecipients[i].MessageId]
			}
		}
	}
	return broadcastRecipients, totalCount, totalPages, nil
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) CountMessageStatusesByBroadcastId(ctx context.Context, broadcastId int32) (map[types.WAMessageStatus]int, error) {
	type row struct {
		Status types.WAMessageStatus `gorm:"column:status"`
		Count  int                   `gorm:"column:count"`
	}
	rows := []row{}
	if err := broadcastRecipientRepository.db.WithContext(ctx).
		Table("customer.broadcast_recipients AS cr").
		Joins("JOIN wa.messages AS m ON m.id = cr.message_id").
		Where("cr.broadcast_id = ?", broadcastId).
		Select("m.status AS status, COUNT(*) AS count").
		Group("m.status").
		Scan(&rows).Error; err != nil {
		broadcastRecipientRepository.logger.ErrorFunction(err, broadcastId)
		return nil, err
	}
	counts := map[types.WAMessageStatus]int{
		types.WAMessageStatusAccepted:  0,
		types.WAMessageStatusSent:      0,
		types.WAMessageStatusDelivered: 0,
		types.WAMessageStatusRead:      0,
		types.WAMessageStatusFailed:    0,
		types.WAMessageStatusRejected:  0,
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) broadcastRecipientAAD(recipient *dao_customer.BroadcastRecipient, purpose string) string {
	return fmt.Sprintf("customer.broadcast_recipients:%s:%s", purpose, recipient.EncryptionID)
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) encryptPayload(recipient *dao_customer.BroadcastRecipient) error {
	if recipient.Payload == nil {
		return nil
	}
	payload, err := json.Marshal(recipient.Payload)
	if err != nil {
		return err
	}
	encrypted, err := encryptStoredSecret(string(payload), broadcastRecipientRepository.broadcastRecipientAAD(recipient, "payload"))
	if err != nil {
		return err
	}
	recipient.PayloadEncrypted = encrypted
	recipient.Payload = nil
	return nil
}

func (broadcastRecipientRepository *BroadcastRecipientRepository) decryptPayload(recipient *dao_customer.BroadcastRecipient) error {
	if recipient.PayloadEncrypted == "" {
		return nil
	}
	payload, err := decryptStoredSecret(recipient.PayloadEncrypted, broadcastRecipientRepository.broadcastRecipientAAD(recipient, "payload"))
	if err != nil {
		broadcastRecipientRepository.logger.ErrorFunction(err, "broadcast_recipient", recipient.Id)
		return err
	}
	return json.Unmarshal([]byte(payload), &recipient.Payload)
}
