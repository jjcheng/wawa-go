package gormdb

import (
	"context"
	"errors"
	"fmt"

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

func (businessPortfolioRepository *WABusinessPortfolioRepository) GetById(ctx context.Context, id int32) (*dao_wa.BusinessPortfolio, error) {
	var businessPortfolio *dao_wa.BusinessPortfolio
	result := businessPortfolioRepository.db.WithContext(ctx).
		Model(&dao_wa.BusinessPortfolio{}).
		Where("id = ?", id).
		First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("BusinessPorfolioRepository.GetById index=0 id=%d error=%w", id, result.Error)
		}
		return nil, result.Error
	}
	if err := businessPortfolioRepository.decryptAccessToken(businessPortfolio); err != nil {
		return nil, fmt.Errorf("BusinessPorfolioRepository.GetById index=0 id=%d error=%w", id, err)
	}
	return businessPortfolio, nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) GetByMetaBusinessPortfolioId(ctx context.Context, metaBusinessPortfolioId string) (*dao_wa.BusinessPortfolio, error) {
	var businessPortfolio *dao_wa.BusinessPortfolio
	result := businessPortfolioRepository.db.WithContext(ctx).Model(&dao_wa.BusinessPortfolio{}).Where("meta_business_portfolio_id = ?", metaBusinessPortfolioId).First(&businessPortfolio)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("BusinessPortfolioRepository.GetByMetaBusinessPortfolioId index=0 metaBusinessPortfolioId=%s error=%w", metaBusinessPortfolioId, result.Error)
		}
		return nil, result.Error
	}
	if err := businessPortfolioRepository.decryptAccessToken(businessPortfolio); err != nil {
		return nil, fmt.Errorf("BusinessPortfolioRepository.GetByMetaBusinessPortfolioId index=1 metaBusinessPortfolioId=%s error=%w", metaBusinessPortfolioId, result.Error)
	}
	return businessPortfolio, nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) Insert(ctx context.Context, businessPortfolio *dao_wa.BusinessPortfolio) error {
	accessToken := businessPortfolio.AccessToken
	if err := businessPortfolioRepository.encryptAccessToken(businessPortfolio); err != nil {
		return fmt.Errorf("BusinessPortfolioRepository.Insert index=0 businessPorfolioId=%d error=%w", businessPortfolio.Id, err)
	}
	defer func() { businessPortfolio.AccessToken = accessToken }()
	err := businessPortfolioRepository.Repository.Insert(ctx, businessPortfolio)
	if err != nil {
		return fmt.Errorf("BusinessPortfolioRepository.Insert index=1 businessPorfolioId=%d error=%w", businessPortfolio.Id, err)
	}
	return nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) Update(ctx context.Context, businessPortfolio *dao_wa.BusinessPortfolio) error {
	accessToken := businessPortfolio.AccessToken
	if err := businessPortfolioRepository.encryptAccessToken(businessPortfolio); err != nil {
		return fmt.Errorf("BusinessPortfolioRepository.Update index=0 businessPorfolioId=%d error=%w", businessPortfolio.Id, err)
	}
	defer func() { businessPortfolio.AccessToken = accessToken }()
	err := businessPortfolioRepository.Repository.Update(ctx, businessPortfolio)
	if err != nil {
		return fmt.Errorf("BusinessPortfolioRepository.Update index=1 businessPorfolioId=%d error=%w", businessPortfolio.Id, err)
	}
	return nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) encryptAccessToken(businessPortfolio *dao_wa.BusinessPortfolio) error {
	if businessPortfolio.AccessToken != "" {
		encrypted, err := encryptSecret(businessPortfolio.AccessToken, businessPortfolioRepository.businessPortfolioAAD(businessPortfolio, "access_token"))
		if err != nil {
			return fmt.Errorf("BusinessPortfolioRepository.encryptAccessToken businessPortfolioId=%d error=%w", businessPortfolio.Id, err)
		}
		businessPortfolio.AccessTokenEncrypted = encrypted
		businessPortfolio.AccessToken = ""
	}
	return nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) decryptAccessToken(businessPortfolio *dao_wa.BusinessPortfolio) error {
	if businessPortfolio.AccessTokenEncrypted == "" {
		return nil
	}
	accessToken, err := decryptSecret(businessPortfolio.AccessTokenEncrypted, businessPortfolioRepository.businessPortfolioAAD(businessPortfolio, "access_token"))
	if err != nil {
		return fmt.Errorf("BusinessPortfolioRepository.decryptAccessToken businessPorfolioId=%d error=%w", businessPortfolio.Id, err)
	}
	businessPortfolio.AccessToken = accessToken
	return nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) businessPortfolioAAD(businessPortfolio *dao_wa.BusinessPortfolio, purpose string) string {
	return fmt.Sprintf("wa.business_portfolios:%s:%s", purpose, businessPortfolio.MetaBusinessPortfolioId)
}
