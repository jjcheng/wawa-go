package repository

import (
	"gorm.io/gorm"
)

type UnitOfWork interface {
	BeginTransaction() UnitOfWork
	Rollback()
	CommitTransaction() error
	DB() *gorm.DB
	// account
	AccountUserRepository() AccountUserRepository
	AccountSettingRepository() AccountSettingRepository
	AccountSessionRepository() AccountSessionRepository
	AccountNotificationRepository() AccountNotificationRepository
	// customer
	CustomerRepository() CustomerRepository
	CampaignRepository() CampaignRepository
	CampaignRecipientRepository() CampaignRecipientRepository
	// wa
	WABusinessPortfolioRepository() WABusinessPortfolioRepository
	WABusinessAccountRepository() WABusinessAccountRepository
	WAPhoneNumberRepository() WAPhoneNumberRepository
	WAHistoryMessageRepository() WAHistoryMessageRepository
	WAMessageRepository() WAMessageRepository
	WAMessageStatusRepository() WAMessageStatusRepository
	WASampleTemplateRepository() WASampleTemplateRepository
}
