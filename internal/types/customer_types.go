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

type BroadcastStatus string

const (
	BroadcastStatusPending   BroadcastStatus = "PENDING"
	BroadcastStatusCompleted BroadcastStatus = "COMPLETED"
	BroadcastStatusSending   BroadcastStatus = "SENDING"
	BroadcastStatusCancelled BroadcastStatus = "CANCELLED"
)

type BroadcastRecipientStatus string

const (
	BroadcastRecipientStatusPending   BroadcastRecipientStatus = "PENDING"
	BroadcastRecipientStatusCompleted BroadcastRecipientStatus = "COMPLETED"
	BroadcastRecipientStatusCancelled BroadcastRecipientStatus = "CANCELLED"
)
