package dto_customer

import (
	dao_customer "github.com/jjcheng/wawa-go/internal/dao/customer"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Customer struct {
	dto.DTOBase
	PhoneNumberId        int32                `json:"phone_number_id"`
	DisplayName          string               `json:"display_name"`
	WADisplayName        string               `json:"wa_display_name"`
	CountryCode          string               `json:"country_code"`
	PhoneNumber          string               `json:"phone_number"`
	MetaUserId           string               `json:"meta_user_id"`
	WAId                 string               `json:"wa_id"`
	Tags                 []string             `json:"tags"`
	Status               types.CustomerStatus `json:"status"`
	Remarks              string               `json:"remarks"`
	AdditionalData       map[string]any       `json:"additional_data"`
	ImportedPhoneNumber  string               `json:"imported_phone_number"`
	Token                string               `json:"token"`
	LatestMessageSending bool                 `json:"latest_message_sending"`
	LatestMessageContent string               `json:"latest_message_content"`
	LatestMessageId      *int32               `json:"latest_message_id"`
	LastMessageTimestamp *int64               `json:"last_message_timestamp"`
}

func NewCustomer(customer dao_customer.Customer) Customer {
	return Customer{
		DTOBase: dto.DTOBase{
			Id:            customer.Id,
			AddedAt:       customer.AddedAt,
			LastUpdatedAt: customer.LastUpdatedAt,
		},
		PhoneNumberId:        customer.PhoneNumberId,
		DisplayName:          customer.DisplayName,
		WADisplayName:        customer.WADisplayName,
		CountryCode:          customer.CountryCode,
		PhoneNumber:          customer.PhoneNumber,
		MetaUserId:           customer.MetaUserId,
		WAId:                 customer.WAId,
		Tags:                 []string(customer.Tags),
		Status:               customer.Status,
		Remarks:              customer.Remarks,
		AdditionalData:       customer.AdditionalData,
		ImportedPhoneNumber:  customer.ImportedPhoneNumber,
		Token:                customer.Token,
		LatestMessageSending: customer.LatestMessageSending,
		LatestMessageContent: customer.LatestMessageContent,
		LatestMessageId:      customer.LatestMessageId,
		LastMessageTimestamp: customer.LastMessageTimestamp,
	}
}
