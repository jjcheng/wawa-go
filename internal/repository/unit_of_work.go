package repository

import (
	"gorm.io/gorm"
)

type UnitOfWork interface {
	BeginTransaction() UnitOfWork
	Rollback()
	CommitTransaction() error
	DB() *gorm.DB
	IsInTransaction() bool
	// account
	AccountUserRepository() AccountUserRepository
	AccountSettingRepository() AccountSettingRepository
	AccountSessionRepository() AccountSessionRepository
	AccountNotificationRepository() AccountNotificationRepository
	AccountCacheRepository() AccountCacheRepository
	AccountUserPhoneNumberRepository() AccountUserPhoneNumberRepository
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
	WABusinessAgentKeywordRepository() WABusinessAgentKeywordRespository
	// commerce
	CommerceCatalogRepository() CommerceCatalogRepository
	CommerceSetRepository() CommerceSetRepository
	CommerceGenericProductRepository() CommerceGenericProductRepository
	CommerceWebsiteRepository() CommerceWebsiteRepository
	CommercePageRepository() CommercePageRepository
	// site
	SiteFeedbackRepository() SiteFeedbackRepository
	// ai
	AIWorkerConversationRepository() AIWorkerConversationRepository
	AIWorkerMessageRepository() AIWorkerMessageRepository
	// ai agent
	AIAgentProfileRepository() AIProfileRepository
	AIAgentFAQRepository() AIAgentFAQRepository
	AIAgentSkillRepository() AIAgentSkillRepository
}
