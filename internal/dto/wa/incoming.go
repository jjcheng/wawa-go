package dto_wa

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Incoming struct {
	Object string          `json:"object"`
	Entry  []IncomingEntry `json:"entry"`
}

type IncomingEntry struct {
	ID      string           `json:"id"`
	Changes []IncomingChange `json:"changes"`
}

type IncomingChange struct {
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
}

type IncomingValue struct {
	MessagingProduct string                `json:"messaging_product"`
	Metadata         IncomingMetadata      `json:"metadata"`
	Contacts         []IncomingContact     `json:"contacts,omitempty"`
	Messages         []IncomingMessage     `json:"messages,omitempty"`
	Standby          *IncomingStandby      `json:"standby,omitempty"`
	Statuses         []Status              `json:"statuses,omitempty"`
	MessageEchoes    []IncomingMessageEcho `json:"message_echoes,omitempty"`
	SyncStatus       string                `json:"sync_status,omitempty"`
	ChunkNumber      int                   `json:"chunk_number,omitempty"`
	// messaing_handovers
	*IncomingHandovers
}

type IncomingMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type IncomingContact struct {
	Profile IncomingProfile `json:"profile"`
	WaID    string          `json:"wa_id,omitempty"` // empty if user opt in for username only
	UserID  string          `json:"user_id"`         // this always return
}

type IncomingProfile struct {
	Name     string `json:"name"`
	UserName string `json:"username,omitempty"` // if user opt in for username only
}

type IncomingMessage struct {
	ID         string         `json:"id"`
	From       string         `json:"from,omitempty"` // empty if user opt in for username only
	FromUserID string         `json:"from_user_id"`   // always return
	Timestamp  string         `json:"timestamp"`
	Type       string         `json:"type"`
	Errors     []StatusError  `json:"errors,omitempty"`
	Payload    map[string]any `json:"-"`
}

type IncomingMessageEcho struct {
	IncomingMessage
	To        string `json:"to"`
	Recipient string `json:"recipient"`
}

type IncomingStandby struct {
	Contacts      []IncomingContact            `json:"contacts"`
	Messages      []IncomingMessage            `json:"messages"`
	MessageEchoes []IncomingMessageStandbyEcho `json:"message_echoes"`
	Statuses      []Status                     `json:"statuses"`
}

type IncomingMessageStandbyEcho struct {
	ID        string                            `json:"id"`
	Timestamp string                            `json:"timestamp"`
	Message   IncomingMessageStandbyEchoMessage `json:"message"`
	Payload   map[string]any                    `json:"payload"`
}

type IncomingMessageStandbyEchoMessage struct {
	To                    string                     `json:"to"`
	Recipient             string                     `json:"recipient"` // metaUserId
	Type                  string                     `json:"type"`
	RecipientType         string                     `json:"recipient_type"` // individual, group
	BizOpaqueCallbackData string                     `json:"biz_opaque_callback_data"`
	Errors                []StatusError              `json:"errors,omitempty"`
	Revoke                *IncomingMessageEchoRevoke `json:"rovoke,omitempty"`
	Edit                  *IncomingMessageEchoEdit   `json:"edit,omitempty"`
}

type IncomingMessageEchoRevoke struct {
	OriginalMessageId string `json:"original_message_id"`
}

type IncomingMessageEchoEdit struct {
	OriginalMessageId string                         `json:"original_message_id"`
	Message           IncomingMessageEchoEditMessage `json:"message"`
}

type IncomingMessageEchoEditMessage struct {
	Context IncomingMessageEchoEditMessageContext `json:"context"`
	Type    string                                `json:"type"`
	Payload map[string]any                        `json:"-"`
}

type IncomingMessageEchoEditMessageContext struct {
	ID string `json:"id"`
}

type IncomingHandovers struct {
	Recipient     IncomingMetadata                `json:"recipient"`
	Sender        IncomingHandoversSender         `json:"sender"`
	Timestamp     string                          `json:"timestamp"`
	Type          string                          `json:"type"`
	ControlPassed *IncomingHandoversControlPassed `json:"control_passed"`
	ControlTaken  *IncomingHandoversControlPassed `json:"control_taken"`
}

type IncomingHandoversSender struct {
	PhoneNumber string `json:"phone_number"`
}

type IncomingHandoversControlPassed struct {
	NewOwnerRole      string `json:"new_owner_role"`
	PreviousOwnerRole string `json:"previous_owner_role"`
}

type IncomingTemplateStatusChange struct {
	Event                   string `json:"event"` // APPROVED
	MessageTemplateId       int64  `json:"message_template_id"`
	MessageTemplateName     string `json:"message_template_name"`
	MessageTemplateLanguage string `json:"message_template_language"`
	Reason                  string `json:"name"`
	MessageTemplateCategory string `json:"message_template_category"`
}

type IncomingItemsBatch struct {
	CatalogId string `json:"catalog_id"`
	Handle    string `json:"handle"`
	Status    string `json:"status"`
}

type IncomingProductFeed struct {
	CatalogId     string `json:"catalog_id"`
	ProductFeedId string `json:"product_feed_id"`
	Status        string `json:"status"`
}

func (message *IncomingMessage) UnmarshalJSON(data []byte) error {
	type incomingMessageAlias IncomingMessage
	var decoded incomingMessageAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	*message = IncomingMessage(decoded)
	message.Payload = payload
	return nil
}

func (message *IncomingMessageEcho) UnmarshalJSON(data []byte) error {
	type incomingMessageAlias IncomingMessageEcho
	var decoded incomingMessageAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	*message = IncomingMessageEcho(decoded)
	message.Payload = payload
	return nil
}

type IncomingMessageText struct {
	Body string `json:"body"`
}

type Status struct {
	ID                    string              `json:"id"`
	Status                string              `json:"status"`
	Timestamp             string              `json:"timestamp"`
	RecipientID           string              `json:"recipient_id"`
	RecipientUserID       string              `json:"recipient_user_id"`
	BizOpaqueCallbackData string              `json:"biz_opaque_callback_data"`
	Conversation          *StatusConversation `json:"conversation,omitempty"`
	Pricing               *StatusPricing      `json:"pricing,omitempty"`
	Errors                []StatusError       `json:"errors,omitempty"`
	Payload               map[string]any      `json:"-"`
}

// this is the value itself
type TemplateStatus struct {
	Event                   string  `json:"event"` // status
	MessageTemplateId       string  `json:"message_template_id"`
	MessageTemplateName     string  `json:"message_template_name"`
	MessageTemplateLanguage string  `json:"message_template_language"`
	Reason                  *string `json:"reason"`
	MessageTemplateCategory string  `json:"message_template_category"`
}

func (status *Status) UnmarshalJSON(data []byte) error {
	type statusAlias Status
	var decoded statusAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	*status = Status(decoded)
	status.Payload = payload
	return nil
}

type StatusConversation struct {
	ID                  string                    `json:"id"`
	ExpirationTimestamp string                    `json:"expiration_timestamp,omitempty"`
	Origin              *StatusConversationOrigin `json:"origin,omitempty"`
}

type StatusConversationOrigin struct {
	Type string `json:"type,omitempty"`
}

type StatusPricing struct {
	Billable     bool   `json:"billable,omitempty"`
	PricingModel string `json:"pricing_model,omitempty"`
	Category     string `json:"category,omitempty"`
	Type         string `json:"type,omitempty"`
}

type StatusError struct {
	Code      int              `json:"code"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	ErrorData *StatusErrorData `json:"error_data,omitempty"`
	Href      string           `json:"href,omitempty"`
}

type StatusErrorData struct {
	Details string `json:"details"`
}

func (statusError *StatusError) Error() string {
	var errors []string
	if statusError.ErrorData != nil {
		errors = append(errors, statusError.ErrorData.Details)
	}
	if len(errors) == 0 {
		errors = append(errors, statusError.Title)
		if statusError.Message != "" && statusError.Message != statusError.Title {
			errors = append(errors, statusError.Message)
		}
	}
	if statusError.Href != "" {
		errors = append(errors, statusError.Href)
	}
	return fmt.Sprintf("WhatsApp API error:\n%s", strings.Join(errors, "\n"))
}
