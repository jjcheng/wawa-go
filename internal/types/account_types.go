package types

type UserType string

const (
	UserTypeMaster   UserType = "MASTER"   // can do everything
	UserTypeOperator UserType = "OPERATOR" // can send messages
	UserTypeAccount  UserType = "ACCOUNT"  // can view usage and cost
)

var UserTypes = []UserType{UserTypeMaster, UserTypeOperator, UserTypeAccount}

type UserStatus string

const (
	UserStatusActive          UserStatus = "ACTIVE"
	UserStatusPendingPassword UserStatus = "PENDING_PASSWORD"
	UserStatusInactive        UserStatus = "INACTIVE"
)
