package types

type OrderCustomersType string

const (
	OrderCustomersTypeFromNew OrderCustomersType = "NEW"
	OrderCustomersTypeFromOld OrderCustomersType = "OLD"
)

type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "ACTIVE"
	CustomerStatusInactive CustomerStatus = "INACTIVE"
)

type CustomerAdditionalDataType string

const (
	CustomerAdditionalDataTypeBirthday CustomerAdditionalDataType = "BIRTHDAY"
)

type CampaignStatus string

const (
	CampaignStatusPending   CampaignStatus = "PENDING"
	CampaignStatusCompleted CampaignStatus = "COMPLETED"
	CampaignStatusSending   CampaignStatus = "SENDING"
	CampaignStatusCancelled CampaignStatus = "CANCELLED"
)

type CampaignRecipientStatus string

const (
	CampaignRecipientStatusPending   CampaignRecipientStatus = "PENDING"   // ready or scheduled, not yet claimed.
	CampaignRecipientStatusCompleted CampaignRecipientStatus = "COMPLETED" // createMessage is called, status not cared
	CampaignRecipientStatusCancelled CampaignRecipientStatus = "CANCELLED" // skipped because parent campaign was cancelled before Meta accepted it
)
