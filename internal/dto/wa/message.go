package dto_wa

import (
	"strings"
	"time"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dto.DTOBase
	Sending       bool                  `json:"sending"`
	PhoneNumberId int32                 `json:"phone_number_id"`
	CustomerId    int32                 `json:"customer_id"`
	WAMessageId   string                `json:"wa_message_id"`
	Timestamp     int64                 `json:"timestamp"`
	Type          string                `json:"type"`
	Status        types.WAMessageStatus `json:"status"`
	Payload       map[string]any        `json:"payload"`
	AttachmentURL string                `json:"attachment_url,omitempty"`
	Billable      bool                  `json:"billable"`
	Category      string                `json:"category,omitempty"`
	BillingType   string                `json:"billing_type,omitempty"`
	Atempts       int32                 `json:"attempts"`
	NextAttemptAt *time.Time            `json:"next_attempt_at"`
	ErrorMessage  string                `json:"error_message"`
	Token         string                `json:"token"`
	ByAgent       bool                  `json:"by_agent"`
	// lazy loaded
	Statuses            []MessageStatus `json:"statuses,omitempty"`
	PreviewHTML         string          `json:"preview_html,omitempty"`
	PreviewDarkHTML     string          `json:"preview_dark_html,omitempty"`
	CustomerName        string          `json:"customer_name"`
	NotificationContent string          `json:"notification_content"`
}

func NewMessage(message dao_wa.Message) Message {
	d := Message{
		DTOBase: dto.DTOBase{
			Id:            message.Id,
			AddedAt:       message.AddedAt,
			LastUpdatedAt: message.LastUpdatedAt,
		},
		Sending:       message.Sending,
		Timestamp:     message.Timestamp,
		PhoneNumberId: message.PhoneNumberId,
		CustomerId:    message.CustomerId,
		WAMessageId:   message.WAMessageId,
		Type:          message.Type,
		Status:        message.Status,
		Payload:       message.Payload,
		AttachmentURL: message.AttachmentURL,
		Billable:      message.Billable,
		BillingType:   message.BillingType,
		Category:      message.Category,
		Atempts:       message.Attempts,
		ErrorMessage:  message.ErrorMessage,
		Token:         message.Token,
		NextAttemptAt: message.NextAttemptAt,
		ByAgent:       message.ByAgent,
	}
	// reset errorMessage and nextAttemptAt if sent successfully
	if d.Status == types.WAMessageStatusAccepted || d.Status == types.WAMessageStatusDelivered || d.Status == types.WAMessageStatusRead || d.Status == types.WAMessageStatusSent {
		d.ErrorMessage = ""
		d.NextAttemptAt = nil
	}
	d.NotificationContent = NotificationContent(message.Type, message.Payload)
	return d
}

func (message Message) Text() string {
	if !strings.EqualFold(strings.TrimSpace(message.Type), "text") {
		return ""
	}
	return nestedMessageString(message.Payload, "text", "body")
}

func NotificationContent(messageType string, payload map[string]any) string {
	switch strings.ToLower(strings.TrimSpace(messageType)) {
	case "text":
		return nestedMessageString(payload, "text", "body")
	case "reaction":
		return nestedMessageString(payload, "reaction", "emoji")
	case "image":
		return "Image"
	case "document":
		return "Document"
	case "video":
		return "Video"
	case "audio":
		return "Audio"
	default:
		return ""
	}
}

func nestedMessageString(payload map[string]any, key string, nestedKey string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	nested, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	content, _ := nested[nestedKey].(string)
	return content
}
