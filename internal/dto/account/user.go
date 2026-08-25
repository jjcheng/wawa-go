package dto_account

import (
	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dto.DTOBase
	Name        string           `json:"name"`
	PhoneNumber string           `json:"phone_number"`
	Email       string           `json:"email"`
	Description string           `json:"description"`
	Type        types.UserType   `json:"type"`
	Status      types.UserStatus `json:"status"`
}

func NewUser(user dao_account.User) User {
	d := User{
		DTOBase: dto.DTOBase{
			Id:         user.Id,
			EntryDate:  user.EntryDate,
			LastUpdate: user.LastUpdate,
		},
		Name:        user.Name,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
		Description: user.Description,
		Type:        user.Type,
		Status:      user.Status,
	}
	return d
}
