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
	CampaignRecipientStatusPending   CampaignRecipientStatus = "PENDING"
	CampaignRecipientStatusCompleted CampaignRecipientStatus = "COMPLETED"
	CampaignRecipientStatusCancelled CampaignRecipientStatus = "CANCELLED"
)
