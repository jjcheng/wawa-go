package dto_customer

import (
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Customer struct {
	dto.DTOBase
	DisplayName         string               `json:"display_name"`
	CountryCode         string               `json:"country_code"`
	PhoneNumber         string               `json:"phone_number"`
	MetaUserId          string               `json:"meta_user_id"`
	WAId                string               `json:"wa_id"`
	Tags                []string             `json:"tags"`
	Status              types.CustomerStatus `json:"status"`
	Remarks             string               `json:"remarks"`
	AdditionalData      map[string]any       `json:"additional_data"`
	ImportedPhoneNumber string               `json:"imported_phone_number"`
}

func NewCustomer(customer dao_customer.Customer) Customer {
	return Customer{
		DTOBase: dto.DTOBase{
			Id:         customer.Id,
			EntryDate:  customer.EntryDate,
			LastUpdate: customer.LastUpdate,
		},
		DisplayName:         customer.DisplayName,
		CountryCode:         customer.CountryCode,
		PhoneNumber:         customer.PhoneNumber,
		MetaUserId:          customer.MetaUserId,
		WAId:                customer.WAId,
		Tags:                []string(customer.Tags),
		Status:              customer.Status,
		Remarks:             customer.Remarks,
		AdditionalData:      customer.AdditionalData,
		ImportedPhoneNumber: customer.ImportedPhoneNumber,
	}
}
