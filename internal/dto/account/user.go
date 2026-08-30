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
	PhoneNumber       string           `json:"phone_number"`
	Email             string           `json:"email"`
	Description       string           `json:"description"`
	Type              types.UserType   `json:"type"`
	Status            types.UserStatus `json:"status"`
	AccessToken       string           `json:"access_token,omitempty" description:"use this access token for subsequent API calls; returned only on login"`
	AccessTokenExpiry *time.Time       `json:"access_token_expiry,omitempty" description:"access token expiry time"`
}

func NewUser(user dao_account.User) User {
	d := User{
		DTOBase: dto.DTOBase{
			Id:         user.Id,
			EntryDate:  user.EntryDate,
			LastUpdate: user.LastUpdate,
		},
		Name:              user.Name,
		PhoneNumber:       user.PhoneNumber,
		Email:             user.Email,
		Description:       user.Description,
		Type:              user.Type,
		Status:            user.Status,
		AccessTokenExpiry: user.AccessTokenExpiry,
	}
	return d
}
