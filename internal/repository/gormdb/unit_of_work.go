package gormdb

import (
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type UnitOfWork struct {
	db     *gorm.DB
	logger *service.Logger
	// account
	accountUserRepository    repository.AccountUserRepository
	accountSettingRepository repository.AccountSettingRepository
	accountSessionRepository repository.AccountSessionRepository
	// customer
	customerRepository repository.CustomerRepository
	// wa
	waBusinessPortfolioRepository  repository.WABusinessPortfolioRepository
	waBusinessAccountRepository    repository.WABusinessAccountRepository
	waPhoneNumberRepository        repository.WAPhoneNumberRepository
	waUserPhoneNumberRepository    repository.WAUserPhoneNumberRepository
	waHistoryMessageRepository     repository.WAHistoryMessageRepository
	waMessageRepository            repository.WAMessageRepository
	waMessageStatusEventRepository repository.WAMessageStatusEventRepository
	waSampleTemplateRepository     repository.WASampleTemplateRepository
	waCampaignRepository           repository.WACampaignRepository
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
	// customer
	unitOfWork.customerRepository = NewCustomerRepository(db, logger)
	// wa
	unitOfWork.waBusinessPortfolioRepository = NewWABusinessPortfolioRepository(db, logger)
	unitOfWork.waBusinessAccountRepository = NewWABusinessAccountRepository(db, logger)
	unitOfWork.waPhoneNumberRepository = NewWAPhoneNumberRepository(db, logger)
	unitOfWork.waUserPhoneNumberRepository = NewWAUserPhoneNumberRepository(db, logger)
	unitOfWork.waHistoryMessageRepository = NewWAHistoryMessageRepository(db, logger)
	unitOfWork.waMessageRepository = NewWAMessageRepository(db, logger)
	unitOfWork.waMessageStatusEventRepository = NewWAMessageStatusEventRepository(db, logger)
	unitOfWork.waSampleTemplateRepository = NewWASampleTemplateRepository(db, logger)
	unitOfWork.waCampaignRepository = NewWACampaignRepository(db, logger)
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

// customer
func (unitOfWork *UnitOfWork) CustomerRepository() repository.CustomerRepository {
	return unitOfWork.customerRepository
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

func (unitOfWork *UnitOfWork) WAUserPhoneNumberRepository() repository.WAUserPhoneNumberRepository {
	return unitOfWork.waUserPhoneNumberRepository
}

func (unitOfWork *UnitOfWork) WAHistoryMessageRepository() repository.WAHistoryMessageRepository {
	return unitOfWork.waHistoryMessageRepository
}

func (unitOfWork *UnitOfWork) WAMessageRepository() repository.WAMessageRepository {
	return unitOfWork.waMessageRepository
}

func (unitOfWork *UnitOfWork) WAMessageStatusEventRepository() repository.WAMessageStatusEventRepository {
	return unitOfWork.waMessageStatusEventRepository
}

func (unitOfWork *UnitOfWork) WASampleTemplateRepository() repository.WASampleTemplateRepository {
	return unitOfWork.waSampleTemplateRepository
}

func (unitOfWork *UnitOfWork) WACampaignRepository() repository.WACampaignRepository {
	return unitOfWork.waCampaignRepository
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
		transaction.logger.ErrorFunction(err)
		return err
	}
	return nil
}

//end of transaction
