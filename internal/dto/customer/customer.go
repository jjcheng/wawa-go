package dto_customer

import (
	"fmt"

	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Customer struct {
	dto.DTOBase
	DisplayName string   `json:"display_name"`
	CountryCode string   `json:"country_code"`
	PhoneNumber string   `json:"phone_number"`
	MetaUserId  string   `json:"meta_user_id"`
	WAId        string   `json:"wa_id" description:"country_code + phone_number" example:"6590909090"`
	Tags        []string `json:"tags"`
}

func NewCustomer(customer dao_customer.Customer) Customer {
	return Customer{
		DTOBase: dto.DTOBase{
			Id:         customer.Id,
			EntryDate:  customer.EntryDate,
			LastUpdate: customer.LastUpdate,
		},
		DisplayName: customer.DisplayName,
		CountryCode: customer.CountryCode,
		PhoneNumber: customer.PhoneNumber,
		MetaUserId:  customer.MetaUserId,
		WAId:        fmt.Sprintf("%s%s", customer.CountryCode, customer.PhoneNumber),
		Tags:        []string(customer.Tags),
	}
}
