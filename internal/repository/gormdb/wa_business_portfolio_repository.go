package gormdb

import (
	"context"
	"errors"
	"strings"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type WABusinessPortfolioRepository struct {
	db     *gorm.DB
	logger *service.Logger
	repository.Repository[dao_wa.BusinessPortfolio]
}

func NewWABusinessPortfolioRepository(db *gorm.DB, logger *service.Logger) repository.WABusinessPortfolioRepository {
	return &WABusinessPortfolioRepository{
		db:         db,
		logger:     logger,
		Repository: NewRepository[dao_wa.BusinessPortfolio](db, logger),
	}
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) Get(ctx context.Context, id int32) (*dao_wa.BusinessPortfolio, error) {
	var businessPortfolio *dao_wa.BusinessPortfolio
	result := businessPortfolioRepository.db.WithContext(ctx).Model(&dao_wa.BusinessPortfolio{}).Where("id = ?", id).First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessPortfolioRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	return businessPortfolio, nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) GetByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) (*dao_wa.BusinessPortfolio, error) {
	var businessPortfolio *dao_wa.BusinessPortfolio
	result := businessPortfolioRepository.db.WithContext(ctx).Model(&dao_wa.BusinessPortfolio{}).Where("meta_business_portfolio_id = ?", metaBusinessPortfolioId).First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			businessPortfolioRepository.logger.ErrorFunction(result.Error, metaBusinessPortfolioId)
		}
		return nil, result.Error
	}
	return businessPortfolio, nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) ListByMetaBusinessPortfolioIds(ctx context.Context, metaBusinessPortfolioIds []string) ([]dao_wa.BusinessPortfolio, error) {
	ids := make([]string, 0, len(metaBusinessPortfolioIds))
	for _, id := range metaBusinessPortfolioIds {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []dao_wa.BusinessPortfolio{}, nil
	}

	var businessPortfolios []dao_wa.BusinessPortfolio
	result := businessPortfolioRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("meta_business_portfolio_id IN ?", ids).
		Order("id").
		Find(&businessPortfolios)
	if result.Error != nil {
		businessPortfolioRepository.logger.ErrorFunction(result.Error, metaBusinessPortfolioIds)
		return nil, result.Error
	}
	return businessPortfolios, nil
}
