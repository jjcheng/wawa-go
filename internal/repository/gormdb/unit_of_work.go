package gormdb

import (
	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"gorm.io/gorm"
)

type UnitOfWork struct {
	db     *gorm.DB
	logger *service.Logger
	// account
	accountUserRepository    repository.AccountUserRepository
	accountSettingRepository repository.AccountSettingRepository
	accountSessionRepository repository.AccountSessionRepository
	// wa
	waBusinessPortfolioRepository repository.WABusinessPortfolioRepository
	waBusinessAccountRepository   repository.WABusinessAccountRepository
	waPhoneNumberRepository       repository.WAPhoneNumberRepository
	waUserPhoneNumberRepository   repository.WAUserPhoneNumberRepository
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
	// wa
	unitOfWork.waBusinessPortfolioRepository = NewWABusinessPortfolioRepository(db, logger)
	unitOfWork.waBusinessAccountRepository = NewWABusinessAccountRepository(db, logger)
	unitOfWork.waPhoneNumberRepository = NewWAPhoneNumberRepository(db, logger)
	unitOfWork.waUserPhoneNumberRepository = NewWAUserPhoneNumberRepository(db, logger)
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

// transaction
func (unitOfWork *UnitOfWork) BeginTransaction() repository.UnitOfWork {
	db := unitOfWork.db.Begin()
	transaction := NewUnitOfWork(db, unitOfWork.logger)
	return transaction
}

func (transaction *UnitOfWork) Rollback() {
	transaction.db.Rollback()
}

func (transaction *UnitOfWork) CommitTransaction() *exception.Exception {
	if err := transaction.db.Commit().Error; err != nil {
		transaction.logger.ErrorFunction(err)
		return exception.NewException(types.ExceptionTypeDatabase)
	}
	return nil
}

//end of transaction
