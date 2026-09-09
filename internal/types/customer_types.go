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
	CampaignStatusPending    CampaignStatus = "PENDING"
	CampaignStatusCompleted  CampaignStatus = "COMPLETED"
	CampaignStatusProcessing CampaignStatus = "PROCESSING"
	CampaignStatusFailed     CampaignStatus = "FAILED"
	CampaignStatusCancelled  CampaignStatus = "CANCELLED"
)
