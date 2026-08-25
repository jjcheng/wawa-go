package dto_cm

import (
	"time"

	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_core "github.com/jjcheng/wawa-go/internal/dto/core"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dto.DTOBase
	ConversationId   int32                           `json:"conversation_id" val:"required" example:"1"`
	SessionId        string                          `json:"session_id" val:"required" example:"8b2670eb-1c79-4f6c-ac96-a3c899b7ce98"`
	Identifier       string                          `json:"identifier" val:"required" example:"8b2670eb-1c79-4f6c-ac96-a3c899b7ce98"`
	Role             types.ChatMessageRole           `json:"role" val:"required" example:"USER"`
	Content          string                          `json:"content" val:"required" example:"Hello"`
	DateTime         time.Time                       `json:"date_time" val:"required" example:"2025-10-01T10:00:00Z"`
	Attachment       *dto_core.ChatMessageAttachment `json:"attachment,omitempty" description:"image, document, video"`
	Location         *dto_core.ChatMessageLocation   `json:"location,omitempty" description:"location details"`
	Status           string                          `json:"status,omitempty" description:"status of the sent message" example:"sent"`
	StatusId         string                          `json:"status_id,omitempty" description:"status id from mobidesk" example:"500"`
	ParentIdentifier string                          `json:"parent_identifier,omitempty"`
}

func NewMessage(m dao_cm.Message) Message {
	message := Message{
		DTOBase: dto.DTOBase{
			Id:         m.Id,
			EntryDate:  m.EntryDate,
			LastUpdate: m.LastUpdate,
		},
		ConversationId:   m.ConversationId,
		SessionId:        m.SessionId,
		Identifier:       m.Identifier,
		Role:             m.Role,
		Content:          m.Content,
		DateTime:         m.DateTime,
		ParentIdentifier: m.ParentIdentifier,
	}
	if m.AttachmentType != "" {
		attachment := dto_core.NewChatMessageAttachment(m.AttachmentType, m.AttachmentName, m.AttachmentUrl, m.AttachmentCaption)
		message.Attachment = &attachment
	}
	if m.LocationName != "" {
		location := dto_core.NewChatMessageLocation(m.LocationName, "", m.LocationLatitude, m.LocationLongitude)
		message.Location = &location
	}
	return message
}
