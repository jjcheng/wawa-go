package dto_wa

import (
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
)

type EmbeddedSignupResponse struct {
	PhoneNumber PhoneNumber      `json:"phone_number"`
	User        dto_account.User `json:"user"`
}
