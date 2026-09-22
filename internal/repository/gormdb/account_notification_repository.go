package gormdb

import (
	"context"
	"fmt"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type AccountNotificationRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_account.Notification]
}

func NewAccountNotificationRepository(db *gorm.DB, logger *service.Logger) repository.AccountNotificationRepository {
	accountNotificationRepository := AccountNotificationRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_account.Notification](db, logger),
	}
	return &accountNotificationRepository
}

func (accountNotificationRepository *AccountNotificationRepository) ListByIds(ctx context.Context, ids []int32, userId int32) ([]dao_account.Notification, error) {
	if len(ids) == 0 {
		return []dao_account.Notification{}, nil
	}
	var notifications []dao_account.Notification
	if err := accountNotificationRepository.db.WithContext(ctx).
		Model(&dao_account.Notification{}).
		Where("user_id = ? AND id IN ?", userId, ids).
		Find(&notifications).Error; err != nil {
		return nil, fmt.Errorf("AccountNotificationRepository.ListByIds ids=%v userId=%d error=%w", ids, userId, err)
	}
	return notifications, nil
}

func (accountNotificationRepository *AccountNotificationRepository) SetStatusByIds(ctx context.Context, userId int32, ids []int32, read bool) error {
	if len(ids) == 0 {
		return nil
	}
	if err := accountNotificationRepository.db.WithContext(ctx).
		Model(&dao_account.Notification{}).
		Where("user_id = ? AND id IN ?", userId, ids).
		Updates(map[string]any{"read": read}).Error; err != nil {
		return fmt.Errorf("AccountNotificationRepository.SetStatusByIds userId=%d ids=%v read=%v error=%w", userId, ids, read, err)
	}
	return nil
}

func (accountNotificationRepository *AccountNotificationRepository) DeleteByIds(ctx context.Context, ids []int32, userId int32) error {
	if len(ids) == 0 {
		return nil
	}
	if err := accountNotificationRepository.db.WithContext(ctx).
		Where("user_id = ? AND id IN ?", userId, ids).
		Delete(&dao_account.Notification{}).Error; err != nil {
		return fmt.Errorf("AccountNotificationRepository.DeleteByIds ids=%v userId=%d error=%w", ids, userId, err)
	}
	return nil
}

func (accountNotificationRepository *AccountNotificationRepository) GetUnreadCount(ctx context.Context, userId int32) (int, error) {
	var count int64
	if err := accountNotificationRepository.db.WithContext(ctx).
		Model(&dao_account.Notification{}).
		Where("user_id = ? AND read = false", userId).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("AccountNotificationRepository.GetUnreadCount userId=%d error=%w", userId, err)
	}
	return int(count), nil
}

func (accountNotificationRepository *AccountNotificationRepository) ListByUserId(ctx context.Context, userId int32, typ types.NotificationType, read *bool, page int, pageSize int) (notifications []dao_account.Notification, totalCount int, totalPages int, err error) {
	query := accountNotificationRepository.db.WithContext(ctx).
		Model(&dao_account.Notification{}).
		Where("user_id = ?", userId)
	if typ != "" {
		query = query.Where("type = ?", typ)
	}
	if read != nil {
		if *read {
			query = query.Where("read = true")
		} else {
			query = query.Where("read = false")
		}
	}
	var count int64
	if err = query.Count(&count).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("AccountNotificationRepository.ListByUserId index=0 userId=%d typ=%s read=%v page=%d pageSize=%d error=%w", userId, typ, read, page, pageSize, err)
	}
	totalCount = int(count)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	totalPages = (totalCount + pageSize - 1) / pageSize
	if err = query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&notifications).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("AccountNotificationRepository.ListByUserId index=1 userId=%d typ=%s read=%v page=%d pageSize=%d error=%w", userId, typ, read, page, pageSize, err)
	}
	return notifications, totalCount, totalPages, nil
}
