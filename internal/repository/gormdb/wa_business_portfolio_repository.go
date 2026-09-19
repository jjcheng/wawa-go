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
			businessPortfolioRepository.logger.ErrorFunction(result.Error, id)
		}
		return nil, result.Error
	}
	if err := businessPortfolioRepository.decryptAccessToken(businessPortfolio); err != nil {
		return nil, err
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
	if err := businessPortfolioRepository.decryptAccessToken(businessPortfolio); err != nil {
		return nil, err
	}
	return businessPortfolio, nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) Insert(ctx context.Context, businessPortfolio *dao_wa.BusinessPortfolio) error {
	accessToken := businessPortfolio.AccessToken
	if err := businessPortfolioRepository.encryptAccessToken(businessPortfolio); err != nil {
		return err
	}
	defer func() { businessPortfolio.AccessToken = accessToken }()
	return businessPortfolioRepository.Repository.Insert(ctx, businessPortfolio)
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) Update(ctx context.Context, businessPortfolio *dao_wa.BusinessPortfolio) error {
	accessToken := businessPortfolio.AccessToken
	if err := businessPortfolioRepository.encryptAccessToken(businessPortfolio); err != nil {
		return err
	}
	defer func() { businessPortfolio.AccessToken = accessToken }()
	return businessPortfolioRepository.Repository.Update(ctx, businessPortfolio)
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) encryptAccessToken(businessPortfolio *dao_wa.BusinessPortfolio) error {
	if businessPortfolio.AccessToken != "" {
		encrypted, err := encryptStoredSecret(businessPortfolio.AccessToken, businessPortfolioRepository.businessPortfolioAAD(businessPortfolio, "access_token"))
		if err != nil {
			return err
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
	accessToken, err := decryptStoredSecret(businessPortfolio.AccessTokenEncrypted, businessPortfolioRepository.businessPortfolioAAD(businessPortfolio, "access_token"))
	if err != nil {
		businessPortfolioRepository.logger.ErrorFunction(err, "business_portfolio", businessPortfolio.Id)
		return err
	}
	businessPortfolio.AccessToken = accessToken
	return nil
}

func (businessPortfolioRepository *WABusinessPortfolioRepository) businessPortfolioAAD(businessPortfolio *dao_wa.BusinessPortfolio, purpose string) string {
	return fmt.Sprintf("wa.business_portfolios:%s:%s", purpose, businessPortfolio.MetaBusinessPortfolioId)
}
