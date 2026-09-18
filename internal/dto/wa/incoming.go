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
	Field string        `json:"field"`
	Value IncomingValue `json:"value"`
}

type IncomingValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         IncomingMetadata  `json:"metadata"`
	Contacts         []IncomingContact `json:"contacts,omitempty"`
	Messages         []IncomingMessage `json:"messages,omitempty"`
	Statuses         []Status          `json:"statuses,omitempty"`
	SyncStatus       string            `json:"sync_status,omitempty"`
	ChunkNumber      int               `json:"chunk_number,omitempty"`
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
