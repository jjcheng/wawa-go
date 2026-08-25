package dto_core

import (
	"strings"

	"github.com/jjcheng/wawa-go/internal/exception"
)

type ChatMessageAttachment struct {
	Name    string `json:"name" val:"required" description:"name of the attachment" example:"test image"`
	Type    string `json:"type" val:"required" description:"image, video, document" example:"image"`
	Url     string `json:"url" val:"required" description:"url of the attachment" example:"https://www..."`
	Caption string `json:"caption" description:"caption of the attachment" example:"image of the exterior"`
}

func (chatMessageAttachment *ChatMessageAttachment) Validate() []exception.InputException {
	chatMessageAttachment.Name = strings.TrimSpace(chatMessageAttachment.Name)
	chatMessageAttachment.Type = strings.TrimSpace(chatMessageAttachment.Type)
	chatMessageAttachment.Url = strings.TrimSpace(chatMessageAttachment.Url)
	chatMessageAttachment.Caption = strings.TrimSpace(chatMessageAttachment.Caption)
	errors := []exception.InputException{}
	if chatMessageAttachment.Type == "" {
		errors = append(errors, exception.NewInputException("type", "missing type"))
	} else if chatMessageAttachment.Type != "image" && chatMessageAttachment.Type != "video" && chatMessageAttachment.Type != "document" && chatMessageAttachment.Type != "audio" {
		errors = append(errors, exception.NewInputException("type", "type must be one of image, video, audio, document"))
	}
	if chatMessageAttachment.Url == "" {
		errors = append(errors, exception.NewInputException("url", "missing url"))
	}
	if chatMessageAttachment.Caption == "" {
		errors = append(errors, exception.NewInputException("caption", "missing caption"))
	}
	return errors
}

func NewChatMessageAttachment(typ string, name string, url string, caption string) ChatMessageAttachment {
	return ChatMessageAttachment{
		Type:    typ,
		Name:    name,
		Url:     url,
		Caption: caption,
	}
}

func (chatMessageAttachment ChatMessageAttachment) Payload() map[string]any {
	dic := map[string]any{
		"type":    chatMessageAttachment.Type,
		"url":     chatMessageAttachment.Url,
		"caption": chatMessageAttachment.Caption,
	}
	if chatMessageAttachment.Name != "" {
		dic["name"] = chatMessageAttachment.Name
	}
	return dic
}
