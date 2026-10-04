package dao_customer

import (
	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/lib/pq"
)

// 1 customer is linked to 1 phone number, but the the same customer can have
// multiple accounts and talk to different phone numbers
type Customer struct {
	dao.DAOBase
	PhoneNumberId       int32                `gorm:"column:phone_number_id"` // wa_phone_number.id not customer's phone number
	DisplayName         string               `gorm:"column:display_name"`    // name tracked by user
	WADisplayName       string               `gorm:"column:wa_display_name"` // name given by WhatsApp
	CountryCode         string               `gorm:"column:country_code"`
	PhoneNumber         string               `gorm:"-"` // not a db column
	MetaUserId          string               `gorm:"column:meta_user_id"`
	WAId                string               `gorm:"-"` // not a db column
	Tags                pq.StringArray       `gorm:"column:tags;type:text[]"`
	Status              types.CustomerStatus `gorm:"column:status"`
	Remarks             string               `gorm:"column:remarks"`
	AdditionalData      map[string]any       `gorm:"-"`                     // not a db column
	ImportedPhoneNumber string               `gorm:"-"`                     // to prevent duplicate when importing, not a db column
	Token               string               `gorm:"column:token"`          // uuid to identify customer
	FromIncomingMessage bool                 `gorm:"from_incoming_message"` // if it's created from incoming message
	//AgentRunning        bool                 `gorm:"agent_running"`         // if it's controlled by business agent
	AgentEnabled bool `gorm:"agent_enabled"` // if allow business agent to engage
	// encryption
	PhoneNumberEncrypted         string `gorm:"column:phone_number_encrypted"`
	PhoneNumberHash              string `gorm:"column:phone_number_hash"`
	WAIdEncrypted                string `gorm:"column:wa_id_encrypted"`
	WAIdHash                     string `gorm:"column:wa_id_hash"`
	ImportedPhoneNumberEncrypted string `gorm:"column:imported_phone_number_encrypted"`
	ImportedPhoneNumberHash      string `gorm:"column:imported_phone_number_hash"`
	AdditionalDataEncrypted      string `gorm:"column:additional_data_encrypted"`
	// latest message fields are read-only values selected by customer list queries
	LatestMessageId               *int32 `gorm:"column:latest_message_id;->"`
	LastMessageTimestamp          *int64 `gorm:"column:last_message_timestamp;->"`
	LatestMessageSending          bool   `gorm:"column:latest_message_sending;->"`
	LatestMessageType             string `gorm:"column:latest_message_type;->"`
	LatestMessageToken            string `gorm:"column:latest_message_token;->"`
	LatestMessagePayloadEncrypted string `gorm:"column:latest_message_payload_encrypted;->"`
	LatestMessageContent          string `gorm:"-"`
}

func (Customer) TableName() string {
	return "customer.customers"
}

func (customer Customer) Base() dao.DAOBase {
	return customer.DAOBase
}
