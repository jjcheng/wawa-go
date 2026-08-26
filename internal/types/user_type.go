package types

type UserType string

const (
	UserTypeAdmin UserType = "ADMIN" // manage users
	UserTypeStaff UserType = "STAFF" // manage phone numbers
)

var UserTypes = []UserType{UserTypeAdmin, UserTypeStaff}

type UserStatus string

const (
	UserStatusActive          UserStatus = "ACTIVE"
	UserStatusPendingPassword UserStatus = "PENDING_PASSWORD"
	UserStatusInactive        UserStatus = "INACTIVE"
)

type CreateUserSource string

const (
	CreateUserSourceEmbededSignUp CreateUserSource = "EMBEDED_SIGNUP"
	CreateUserSourceAdmin         CreateUserSource = "ADMIN"
)
