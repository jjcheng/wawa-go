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
	AccountCacheRepository() AccountCacheRepository
	// customer
	CustomerRepository() CustomerRepository
	BroadcastRepository() BarodcastRepository
	BroadcastRecipientRepository() BroadcastRecipientRepository
	// wa
	WABusinessPortfolioRepository() WABusinessPortfolioRepository
	WABusinessAccountRepository() WABusinessAccountRepository
	WAPhoneNumberRepository() WAPhoneNumberRepository
	WAHistoryMessageRepository() WAHistoryMessageRepository
	WAMessageRepository() WAMessageRepository
	WAMessageStatusRepository() WAMessageStatusRepository
	WASampleTemplateRepository() WASampleTemplateRepository
	// commerce
	CommerceCatalogRepository() CommerceCatalogRepository
	CommerceSetRepository() CommerceSetRepository
	CommerceGenericProductRepository() CommerceGenericProductRepository
	CommerceWebsiteRepository() CommerceWebsiteRepository
	// site
	SiteFeedbackRepository() SiteFeedbackRepository
}
