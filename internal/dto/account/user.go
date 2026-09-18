package dto_account

import (
	"time"

	dao_account "github.com/jjcheng/wawa-go/internal/dao/account"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/types"
)

type User struct {
	dto.DTOBase
	Name              string           `json:"name"`
	CountryCode       string           `json:"country_code"`
	PhoneNumber       string           `json:"phone_number"`
	Email             string           `json:"email"`
	Description       string           `json:"description"`
	Type              types.UserType   `json:"type"`
	Status            types.UserStatus `json:"status"`
	AccessToken       string           `json:"access_token,omitempty" description:"only returned during login"`
	AccessTokenExpiry *time.Time       `json:"access_token_expiry,omitempty" description:"access token expiry time"`
	// only used in embedded signup if the activate meta function failed
	WAActivated       bool   `json:"wa_activated,omitempty"`
	WAActivationError string `json:"wa_activation_error,omitempty"`
	New               bool   `json:"new" description:"indicate this is a new user, if not, after embedded signup need to re login"`
	// to retrieve all WA related objects
	WA *UserWA `json:"-"`
	// login info
	Session *Session `json:"-"`
}

type UserWA struct {
	PhoneNumber_                 *dto_wa.PhoneNumber       `json:"-"`
	BusinessAccount              *dto_wa.BusinessAccount   `json:"-"`
	BusinessPortfolio            *dto_wa.BusinessPortfolio `json:"-"`
	BusinessPortfolioAccessToken string                    `json:"-"`
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
		Email:       user.Email,
		Description: user.Description,
		Type:        user.Type,
		Status:      user.Status,
	}
	return d
}
