package dto_account

import (
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dto.DTOBase
	Name              string           `json:"name"`
	CountryCode       string           `json:"country_code"`
	PhoneNumber       string           `json:"phone_number"`
	Description       string           `json:"description"`
	Type              types.UserType   `json:"type"`
	Status            types.UserStatus `json:"status"`
	AccessToken       string           `json:"access_token,omitempty"`
	AccessTokenExpiry *time.Time       `json:"access_token_expiry,omitempty" description:"access token expiry time"`
	// only used in embedded signup if the activate meta function failed
	WAActivated       bool   `json:"wa_activated"`
	WAActivationError string `json:"wa_activation_error,omitempty"`
}

func NewUser(user dao_account.User) User {
	d := User{
		DTOBase: dto.DTOBase{
			Id:         user.Id,
			EntryDate:  user.EntryDate,
			LastUpdate: user.LastUpdate,
		},
		Name:        user.Name,
		CountryCode: user.CountryCode,
		PhoneNumber: user.PhoneNumber,
		Description: user.Description,
		Type:        user.Type,
		Status:      user.Status,
	}
	return d
}
