package types

type UserType string

const (
	UserTypeMaster   UserType = "MASTER"   // can do everything
	UserTypeOperator UserType = "OPERATOR" // can send messages
)

var UserTypes = []UserType{UserTypeMaster, UserTypeOperator}

type UserStatus string

const (
	UserStatusActive            UserStatus = "ACTIVE"
	UserStatusPendingPassword   UserStatus = "PENDING_PASSWORD"   // a new user signed up in embedded signup
	UserStatusPendingAssignment UserStatus = "PENDING_ASSIGNMENT" // a MASTER user created a new phone number pending assignment to user
	UserStatusInactive          UserStatus = "INACTIVE"
	UserStatusClosed            UserStatus = "CLOSED"
)

type NotificationType string

const (
	NotificationTypeSuccess NotificationType = "SUCCESS"
	NotificationTypeInfo    NotificationType = "INFO"
	NotificationTypeWarning NotificationType = "WARNING"
	NotificationTypeError   NotificationType = "ERROR"
)

type NotificationCategory string

const (
	NotificationCategoryPending  NotificationCategory = "PENDING"
	NotificationCategoryHandsOff NotificationCategory = "HANDS-OFF"
)

type NotificationIconType string

const (
	NotificationIconTypeCustomer  NotificationIconType = "CUSTOMER"
	NotificationIconTypeBroadcast NotificationIconType = "BROADCAST"
	NotificationIconTypeWebsite   NotificationIconType = "WEBSITE"
	NotificationIconTypeChat      NotificationIconType = "CHAT"
	NotificationIconTypeJoin      NotificationIconType = "JOIN"
	NotificationIconTypeTemplate  NotificationIconType = "TEMPLATE"
	NotificationIconTypeTODO      NotificationIconType = "TODO"
	NotificationIconTypeSuccess   NotificationIconType = "SUCCESS"
	NotificationIconTypeError     NotificationIconType = "ERROR"
	NotificationIconTypeWarning   NotificationIconType = "WARNING"
)
