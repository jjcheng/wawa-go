package types

type UserType string

const (
	UserTypeAdmin   UserType = "ADMIN"   // can do everything
	UserTypeStaff   UserType = "SUPPORT" // can send messages
	UserTypeAccount UserType = "ACCOUNT" // can view usage and cost
)

var UserTypes = []UserType{UserTypeAdmin, UserTypeStaff}

type UserStatus string

const (
	UserStatusActive          UserStatus = "ACTIVE"
	UserStatusPendingPassword UserStatus = "PENDING_PASSWORD"
	UserStatusInactive        UserStatus = "INACTIVE"
)

// type CreateUserSource string

// const (
// 	CreateUserSourceEmbededSignUp CreateUserSource = "EMBEDED_SIGNUP"
// 	CreateUserSourceAdmin         CreateUserSource = "ADMIN"
// )
