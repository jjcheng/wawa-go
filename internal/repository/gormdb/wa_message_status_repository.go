package gormdb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WAMessageStatusRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.MessageStatus]
}

func NewWAMessageStatusRepository(db *gorm.DB, logger *service.Logger) repository.WAMessageStatusRepository {
	return &WAMessageStatusRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.MessageStatus](db, logger),
	}
}

func (messageStatusRepository *WAMessageStatusRepository) Insert(ctx context.Context, messageStatus *dao_wa.MessageStatus) error {
	if messageStatus.EncryptionID == "" {
		messageStatus.EncryptionID = uuid.NewString()
	}
	payload := messageStatus.Payload
	if err := messageStatusRepository.encryptPayload(messageStatus); err != nil {
		return err
	}
	defer func() { messageStatus.Payload = payload }()
	return messageStatusRepository.Repository.Insert(ctx, messageStatus)
}

func (messageStatusRepository *WAMessageStatusRepository) Update(ctx context.Context, messageStatus *dao_wa.MessageStatus) error {
	payload := messageStatus.Payload
	if err := messageStatusRepository.encryptPayload(messageStatus); err != nil {
		return err
	}
	defer func() { messageStatus.Payload = payload }()
	return messageStatusRepository.Repository.Update(ctx, messageStatus)
}

func (messageStatusRepository *WAMessageStatusRepository) GetById(ctx context.Context, id int32) (*dao_wa.MessageStatus, error) {
	var messageStatus dao_wa.MessageStatus
	if err := messageStatusRepository.db.WithContext(ctx).
		Where("id = ?", id).
		First(&messageStatus).Error; err != nil {
		return nil, err
	}
	if err := messageStatusRepository.decryptPayload(&messageStatus); err != nil {
		return nil, err
	}
	return &messageStatus, nil
}

func (messageStatusRepository *WAMessageStatusRepository) ListByMessageId(ctx context.Context, messageId int32) ([]dao_wa.MessageStatus, error) {
	var events []dao_wa.MessageStatus
	result := messageStatusRepository.db.WithContext(ctx).
		Model(&dao_wa.MessageStatus{}).
		Where("message_id = ?", messageId).
		Order("timestamp").
		Find(&events)
	if result.Error != nil {
		messageStatusRepository.logger.ErrorFunction(result.Error, messageId)
		return nil, result.Error
	}
	for i := range events {
		if err := messageStatusRepository.decryptPayload(&events[i]); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (messageStatusRepository *WAMessageStatusRepository) encryptPayload(messageStatus *dao_wa.MessageStatus) error {
	if messageStatus.Payload == nil {
		return nil
	}
	payload, err := json.Marshal(messageStatus.Payload)
	if err != nil {
		return err
	}
	encrypted, err := encryptStoredSecret(string(payload), messageStatusRepository.messageStatusAAD(messageStatus, "payload"))
	if err != nil {
		return err
	}
	messageStatus.PayloadEncrypted = encrypted
	messageStatus.Payload = nil
	return nil
}

func (messageStatusRepository *WAMessageStatusRepository) decryptPayload(messageStatus *dao_wa.MessageStatus) error {
	if messageStatus.PayloadEncrypted == "" {
		return nil
	}
	payload, err := decryptStoredSecret(messageStatus.PayloadEncrypted, messageStatusRepository.messageStatusAAD(messageStatus, "payload"))
	if err != nil {
		messageStatusRepository.logger.ErrorFunction(err, "message_status", messageStatus.Id)
		return err
	}
	return json.Unmarshal([]byte(payload), &messageStatus.Payload)
}

func (messageStatusRepository *WAMessageStatusRepository) messageStatusAAD(messageStatus *dao_wa.MessageStatus, purpose string) string {
	return fmt.Sprintf("wa.message_status:%s:%s", purpose, messageStatus.EncryptionID)
}
