package dto_customer

import (
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
)

type Customer struct {
	dto.DTOBase
	DisplayName string   `json:"display_name"`
	CountryCode string   `json:"country_code"`
	PhoneNumber string   `json:"phone_number"`
	BSUID       string   `json:"bsuid"`
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
		BSUID:       customer.BSUID,
		Tags:        []string(customer.Tags),
	}
}
