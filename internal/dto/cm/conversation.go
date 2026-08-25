package dto_cm

import (
	dao_cm "github.com/jjcheng/wawa-go/internal/dao/cm"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_core "github.com/jjcheng/wawa-go/internal/dto/core"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Conversation struct {
	dto.DTOBase
	Title      string                   `json:"title" val:"required" example:"Top 10 best selling projects"`
	Identifier string                   `json:"identifier" val:"required" example:"8b2670eb-1c79-4f6c-ac96-a3c899b7ce98"`
	Channel    types.ChatMessageChannel `json:"channel" val:"required" example:"WHATSAPP"`
	Source     string                   `json:"source" val:"required" example:"campaign://1"`
	// get from conversation messages
	LastMessage *dto_core.ChatMessage `json:"last_message,omitempty" description:"the last interaction"`
	// used only in start to determine if a full message retrieval is needed
	New bool `json:"-"`
}

func NewConversation(conversation dao_cm.Conversation) Conversation {
	conv := Conversation{
		DTOBase: dto.DTOBase{
			Id:         conversation.Id,
			EntryDate:  conversation.EntryDate,
			LastUpdate: conversation.LastUpdate,
		},
		Title:      conversation.Title,
		Identifier: conversation.Identifier,
		Channel:    conversation.Channel,
		Source:     conversation.Source,
	}
	return conv
}
