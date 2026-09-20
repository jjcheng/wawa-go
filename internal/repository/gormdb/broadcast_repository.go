package gormdb

import (
	"context"
	"math"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
	"gorm.io/gorm"
)

type BroadcastRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_customer.Broadcast]
}

func NewBroadcastRepository(db *gorm.DB, logger *service.Logger) repository.BarodcastRepository {
	return &BroadcastRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_customer.Broadcast](db, logger),
	}
}

func (broadcastRepository *BroadcastRepository) ListByUserId(ctx context.Context, userId int32, name string, status types.BroadcastStatus, page int, pageSize int) (*dto.ListResponse[dao_customer.Broadcast], error) {
	var broadcasts []dao_customer.Broadcast
	query := broadcastRepository.db.WithContext(ctx).
		Model(&dao_customer.Broadcast{}).
		Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	var numberOfItems int64
	if err := query.Count(&numberOfItems).Error; err != nil {
		broadcastRepository.logger.ErrorFunction(err, userId, status, page, pageSize)
		return nil, err
	}
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&broadcasts).Error; err != nil {
		broadcastRepository.logger.ErrorFunction(err, userId, status, page, pageSize)
		return nil, err
	}
	numberOfPages := int(math.Ceil(float64(numberOfItems) / float64(pageSize)))
	result := dto.NewPagedListResponse(broadcasts, numberOfPages, int(numberOfItems))
	return &result, nil
}

func (broadcastRepository *BroadcastRepository) CheckNameExist(ctx context.Context, userId int32, name string) (bool, error) {
	var count int64
	err := broadcastRepository.db.WithContext(ctx).
		Model(&dao_customer.Broadcast{}).
		Where("user_id = ? AND name = ?", userId, name).
		Count(&count).Error
	if err != nil {
		broadcastRepository.logger.ErrorFunction(err, userId, name)
		return false, err
	}
	return count > 0, nil
}

func (broadcastRepository *BroadcastRepository) ListByIds(ctx context.Context, ids []int32) ([]dao_customer.Broadcast, error) {
	if len(ids) == 0 {
		return []dao_customer.Broadcast{}, nil
	}
	var broadcasts []dao_customer.Broadcast
	result := broadcastRepository.db.WithContext(ctx).
		Model(&dao_customer.Broadcast{}).
		Where("id IN ?", ids).
		Order("id").
		Find(&broadcasts)
	if result.Error != nil {
		broadcastRepository.logger.ErrorFunction(result.Error, ids)
		return nil, result.Error
	}
	return broadcasts, nil
}

func (broadcastRepository *BroadcastRepository) ListByMessageIds(ctx context.Context, messageIds []int32) ([]dao_customer.Broadcast, error) {
	if len(messageIds) == 0 {
		return []dao_customer.Broadcast{}, nil
	}
	var broadcasts []dao_customer.Broadcast
	result := broadcastRepository.db.WithContext(ctx).
		Table("customer.broadcast AS c").
		Joins("JOIN customer.broadcast_recipients AS cr ON cr.broadcast_id = c.id").
		Where("cr.message_id IN ?", messageIds).
		Distinct("c.*").
		Find(&broadcasts)
	if result.Error != nil {
		broadcastRepository.logger.ErrorFunction(result.Error, messageIds)
		return nil, result.Error
	}
	return broadcasts, nil
}

func (broadcastRepository *BroadcastRepository) ListPendingBroadcasts(ctx context.Context) ([]dao_customer.Broadcast, error) {
	var broadcasts []dao_customer.Broadcast
	result := broadcastRepository.db.WithContext(ctx).
		Model(&dao_customer.Broadcast{}).
		Where("status = ? AND send_date <= NOW()", types.BroadcastStatusPending).
		Order("send_date, id").
		Find(&broadcasts)
	if result.Error != nil {
		broadcastRepository.logger.ErrorFunction(result.Error)
		return nil, result.Error
	}
	return broadcasts, nil
}
