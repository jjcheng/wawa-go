package gormdb

import (
	"context"
	"errors"
	"fmt"

	dao_commerce "github.com/jjcheng/wawa-go/internal/dao/commerce"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type CommercePageRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_commerce.Page]
}

func NewCommercePageRepository(db *gorm.DB, logger *service.Logger) repository.CommercePageRepository {
	return &CommercePageRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_commerce.Page](db, logger),
	}
}

func (commercePageRepository *CommercePageRepository) ListByWebsiteId(ctx context.Context, websiteId int32) ([]dao_commerce.Page, error) {
	var pages []dao_commerce.Page
	if err := commercePageRepository.db.WithContext(ctx).
		Model(&dao_commerce.Page{}).
		Where("website_id = ?", websiteId).
		Order("rank, id").
		Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("CommercePageRepository.ListByWebsiteId websiteId=%d error=%w", websiteId, err)
	}
	return pages, nil
}

func (commercePageRepository *CommercePageRepository) GetPagesCountByWebsiteId(ctx context.Context, websiteId int32) (int, error) {
	var count int64
	if err := commercePageRepository.db.WithContext(ctx).
		Model(&dao_commerce.Page{}).
		Where("website_id = ?", websiteId).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("CommercePageRepository.GetPagesCountByWebsiteId websiteId=%d error=%w", websiteId, err)
	}
	return int(count), nil
}

func (commercePageRepository *CommercePageRepository) ListByOnNavBar(ctx context.Context, websiteId int32) ([]dao_commerce.Page, error) {
	var pages []dao_commerce.Page
	if err := commercePageRepository.db.WithContext(ctx).
		Model(&dao_commerce.Page{}).
		Where("website_id = ? AND nav = true", websiteId).
		Order("rank, id").Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("CommercePageRepository.ListByOnNavBar websiteId=%d error=%w", websiteId, err)
	}
	return pages, nil
}

func (commercePageRepository *CommercePageRepository) GetBySlug(ctx context.Context, websiteId int32, slug string) (*dao_commerce.Page, error) {
	var page dao_commerce.Page
	if err := commercePageRepository.db.WithContext(ctx).
		Where("website_id = ? AND slug = ?", websiteId, slug).
		First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("CommercePageRepository.GetBySlug websiteId=%d slug=%s error=%w", websiteId, slug, err)
	}
	return &page, nil
}
