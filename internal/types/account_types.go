package types

type UserType string

const (
	UserTypeMaster   UserType = "MASTER"   // can do everything
	UserTypeOperator UserType = "OPERATOR" // can send messages
)

var UserTypes = []UserType{UserTypeMaster, UserTypeOperator}

type UserStatus string

const (
	UserStatusActive          UserStatus = "ACTIVE"
	UserStatusPendingPassword UserStatus = "PENDING_PASSWORD"
	UserStatusInactive        UserStatus = "INACTIVE"
	UserStatusClosed          UserStatus = "CLOSED"
)

type NotificationType string

const (
	NotificationTypeSuccess NotificationType = "SUCCESS"
	NotificationTypeInfo    NotificationType = "INFO"
	NotificationTypeWarning NotificationType = "WARNING"
	NotificationTypeError   NotificationType = "ERROR"
)
