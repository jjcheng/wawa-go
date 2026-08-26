package repository

import (
	"github.com/jjcheng/wawa-go/internal/exception"

	"gorm.io/gorm"
)

type UnitOfWork interface {
	BeginTransaction() UnitOfWork
	Rollback()
	CommitTransaction() *exception.Exception
	DB() *gorm.DB
	// account
	AccountUserRepository() AccountUserRepository
	AccountSettingRepository() AccountSettingRepository
	AccountSessionRepository() AccountSessionRepository
	// wa
	WABusinessPortfolioRepository() WABusinessPortfolioRepository
	WABusinessAccountRepository() WABusinessAccountRepository
	WAPhoneNumberRepository() WAPhoneNumberRepository
	WAUserPhoneNumberRepository() WAUserPhoneNumberRepository
}
