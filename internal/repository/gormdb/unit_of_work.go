package gormdb

import (
	"fmt"

	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type UnitOfWork struct {
	db     *gorm.DB
	logger *service.Logger
	// account
	accountUserRepository         repository.AccountUserRepository
	accountSettingRepository      repository.AccountSettingRepository
	accountSessionRepository      repository.AccountSessionRepository
	accountNotificationRepository repository.AccountNotificationRepository
	accountCacheRepository        repository.AccountCacheRepository
	// customer
	customerRepository           repository.CustomerRepository
	broadcastRepository          repository.BarodcastRepository
	broadcastRecipientRepository repository.BroadcastRecipientRepository
	// wa
	waBusinessPortfolioRepository repository.WABusinessPortfolioRepository
	waBusinessAccountRepository   repository.WABusinessAccountRepository
	waPhoneNumberRepository       repository.WAPhoneNumberRepository
	waHistoryMessageRepository    repository.WAHistoryMessageRepository
	waMessageRepository           repository.WAMessageRepository
	waMessageStatusRepository     repository.WAMessageStatusRepository
	waSampleTemplateRepository    repository.WASampleTemplateRepository
	// commerce
	commerceCatalogRepository        repository.CommerceCatalogRepository
	commerceSetRepository            repository.CommerceSetRepository
	commerceGenericProductRepository repository.CommerceGenericProductRepository
	commerceWebsiteRepository        repository.CommerceWebsiteRepository
	commercePageRepository           repository.CommercePageRepository
	// site
	siteFeedbackRepository repository.SiteFeedbackRepository
}

func NewUnitOfWork(db *gorm.DB, logger *service.Logger) repository.UnitOfWork {
	unitOfWork := UnitOfWork{
		db:     db,
		logger: logger,
	}
	// account
	unitOfWork.accountUserRepository = NewAccountUserRepository(db, logger)
	unitOfWork.accountSettingRepository = NewAccountSettingRepository(db, logger)
	unitOfWork.accountSessionRepository = NewAccountSessionRepository(db, logger)
	unitOfWork.accountNotificationRepository = NewAccountNotificationRepository(db, logger)
	unitOfWork.accountCacheRepository = NewAccountCacheRepository(db, logger)
	// customer
	unitOfWork.customerRepository = NewCustomerRepository(db, logger)
	unitOfWork.broadcastRepository = NewBroadcastRepository(db, logger)
	unitOfWork.broadcastRecipientRepository = NewBroadcastRecipientRepository(db, logger)
	// wa
	unitOfWork.waBusinessPortfolioRepository = NewWABusinessPortfolioRepository(db, logger)
	unitOfWork.waBusinessAccountRepository = NewWABusinessAccountRepository(db, logger)
	unitOfWork.waPhoneNumberRepository = NewWAPhoneNumberRepository(db, logger)
	unitOfWork.waHistoryMessageRepository = NewWAHistoryMessageRepository(db, logger)
	unitOfWork.waMessageRepository = NewWAMessageRepository(db, logger)
	unitOfWork.waMessageStatusRepository = NewWAMessageStatusRepository(db, logger)
	unitOfWork.waSampleTemplateRepository = NewWASampleTemplateRepository(db, logger)
	// commerce
	unitOfWork.commerceCatalogRepository = NewCommerceCatalogRepository(db, logger)
	unitOfWork.commerceSetRepository = NewCommerceSetRepository(db, logger)
	unitOfWork.commerceGenericProductRepository = NewCommericeGenericProductRepository(db, logger)
	unitOfWork.commerceWebsiteRepository = NewCommerceWebsiteRepository(db, logger)
	unitOfWork.commercePageRepository = NewCommercePageRepository(db, logger)
	// site
	unitOfWork.siteFeedbackRepository = NewSiteFeedbackRepository(db, logger)
	return &unitOfWork
}

func (unitOfWork *UnitOfWork) DB() *gorm.DB {
	return unitOfWork.db
}

// account
func (unitOfWork *UnitOfWork) AccountUserRepository() repository.AccountUserRepository {
	return unitOfWork.accountUserRepository
}

func (unitOfWork *UnitOfWork) AccountSettingRepository() repository.AccountSettingRepository {
	return unitOfWork.accountSettingRepository
}

func (unitOfWork *UnitOfWork) AccountSessionRepository() repository.AccountSessionRepository {
	return unitOfWork.accountSessionRepository
}

func (unitOfWork *UnitOfWork) AccountNotificationRepository() repository.AccountNotificationRepository {
	return unitOfWork.accountNotificationRepository
}

func (unitOfWork *UnitOfWork) AccountCacheRepository() repository.AccountCacheRepository {
	return unitOfWork.accountCacheRepository
}

// customer
func (unitOfWork *UnitOfWork) CustomerRepository() repository.CustomerRepository {
	return unitOfWork.customerRepository
}

func (unitOfWork *UnitOfWork) BroadcastRepository() repository.BarodcastRepository {
	return unitOfWork.broadcastRepository
}

func (unitOfWork *UnitOfWork) BroadcastRecipientRepository() repository.BroadcastRecipientRepository {
	return unitOfWork.broadcastRecipientRepository
}

// wa
func (unitOfWork *UnitOfWork) WABusinessPortfolioRepository() repository.WABusinessPortfolioRepository {
	return unitOfWork.waBusinessPortfolioRepository
}

func (unitOfWork *UnitOfWork) WABusinessAccountRepository() repository.WABusinessAccountRepository {
	return unitOfWork.waBusinessAccountRepository
}

func (unitOfWork *UnitOfWork) WAPhoneNumberRepository() repository.WAPhoneNumberRepository {
	return unitOfWork.waPhoneNumberRepository
}

func (unitOfWork *UnitOfWork) WAHistoryMessageRepository() repository.WAHistoryMessageRepository {
	return unitOfWork.waHistoryMessageRepository
}

func (unitOfWork *UnitOfWork) WAMessageRepository() repository.WAMessageRepository {
	return unitOfWork.waMessageRepository
}

func (unitOfWork *UnitOfWork) WAMessageStatusRepository() repository.WAMessageStatusRepository {
	return unitOfWork.waMessageStatusRepository
}

func (unitOfWork *UnitOfWork) WASampleTemplateRepository() repository.WASampleTemplateRepository {
	return unitOfWork.waSampleTemplateRepository
}

// commerce
func (unitOfWork *UnitOfWork) CommerceCatalogRepository() repository.CommerceCatalogRepository {
	return unitOfWork.commerceCatalogRepository
}

func (unitOfWork *UnitOfWork) CommerceSetRepository() repository.CommerceSetRepository {
	return unitOfWork.commerceSetRepository
}

func (unitOfWork *UnitOfWork) CommerceGenericProductRepository() repository.CommerceGenericProductRepository {
	return unitOfWork.commerceGenericProductRepository
}

func (unitOfWork *UnitOfWork) CommerceWebsiteRepository() repository.CommerceWebsiteRepository {
	return unitOfWork.commerceWebsiteRepository
}

func (unitOfWork *UnitOfWork) CommercePageRepository() repository.CommercePageRepository {
	return unitOfWork.commercePageRepository
}

// site
func (unitOfWork *UnitOfWork) SiteFeedbackRepository() repository.SiteFeedbackRepository {
	return unitOfWork.siteFeedbackRepository
}

// transaction
func (unitOfWork *UnitOfWork) BeginTransaction() repository.UnitOfWork {
	db := unitOfWork.db.Begin()
	transaction := NewUnitOfWork(db, unitOfWork.logger)
	return transaction
}

func (transaction *UnitOfWork) Rollback() {
	transaction.db.Rollback()
}

func (transaction *UnitOfWork) CommitTransaction() error {
	if err := transaction.db.Commit().Error; err != nil {
		return fmt.Errorf("UnitOfWork.CommitTransaction error=%w", err)
	}
	return nil
}

//end of transaction
