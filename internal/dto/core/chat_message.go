package dto_core

import (
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
)

type ChatMessage struct {
	Identifier       string                 `json:"identifier" val:"required" description:"uuid to track (mostly for mobi)"`
	Content          string                 `json:"content" val:"required" description:"content of the message" example:"hi"`
	DateTime         time.Time              `json:"date_time" val:"required" description:"date time of the message creation" example:"2025-10-01T10:00:00Z"`
	Attachment       *ChatMessageAttachment `json:"attachment" description:"image, document, video"`
	Location         *ChatMessageLocation   `json:"location" description:"location with coordinate and name"`
	ParentIdentifier string                 `json:"parent_identifier" description:"parent message identifier"`
	New              bool                   `json:"-"`
	Replied          bool                   `json:"-"` // when sending message, don't update replied more than once
	Sent             bool                   `json:"-"` // sometimes the message is already sent
}

// must start with http:// or https:// and is a valid url and end with .png, .jpg, .jpeg, .webp
func (chatMessage *ChatMessage) CheckIsImage() bool {
	isImage, _ := helper.CheckIsImage(chatMessage.Content)
	return isImage
}

// func GetLastestMessageWithRole(messages []ChatMessage, role types.ChatMessageRole) *ChatMessage {
// 	if len(messages) == 0 {
// 		return nil
// 	}
// 	for i := len(messages) - 1; i >= 0; i-- {
// 		if messages[i].Role == role {
// 			return &messages[i]
// 		}
// 	}
// 	return nil
// }

// func GetLastestNewMessageWithRole(messages []ChatMessage, role types.ChatMessageRole) *ChatMessage {
// 	if len(messages) == 0 {
// 		return nil
// 	}
// 	for i := len(messages) - 1; i >= 0; i-- {
// 		if messages[i].New && messages[i].Role == role {
// 			return &messages[i]
// 		}
// 	}
// 	return nil
// }

// func GetNewMessagesWithRole(messages []ChatMessage, role types.ChatMessageRole) []ChatMessage {
// 	return helper.Filter(messages, func(cm ChatMessage) bool {
// 		return cm.New && cm.Role == role
// 	})
// }

// // includes the last message of the role
// func GetMessagesSinceRole(messages []ChatMessage, role types.ChatMessageRole) []ChatMessage {
// 	if len(messages) == 0 {
// 		return nil
// 	}
// 	roleMessages := []ChatMessage{}
// 	for i := len(messages) - 1; i >= 0; i-- {
// 		roleMessages = append(roleMessages, messages[i])
// 		if messages[i].Role == role {
// 			break
// 		}
// 	}
// 	helper.Reverse(roleMessages)
// 	return roleMessages
// }

func ValidateChatMessages(messages *[]ChatMessage) []exception.InputException {
	errors := []exception.InputException{}
	// at least 1 message
	if len(*messages) == 0 {
		errors = append(errors, exception.NewInputException("messages", "missing messages"))
		return errors
	}
	for i, message := range *messages {
		// remove illegal characters
		(*messages)[i].Content = strings.ReplaceAll((*messages)[i].Content, "’", "'")
		if strings.TrimSpace(message.Content) == "" {
			continue
			// errors = append(errors, exception.NewInputException("messages", fmt.Sprintf("messages[%d].content is empty", i)))
		}
	}
	return errors
}

// // new: true, identifier: new uuid, dateTime: now
// func NewChatMessage(role types.ChatMessageRole, content string) ChatMessage {
// 	return ChatMessage{
// 		Role:       role,
// 		Content:    content,
// 		DateTime:   time.Now(),
// 		New:        true,
// 		Identifier: uuid.NewString(),
// 	}
// }

// func NewChatMessageWithAttachment(role types.ChatMessageRole, content string, attachment *ChatMessageAttachment) ChatMessage {
// 	return ChatMessage{
// 		Role:       role,
// 		Content:    content,
// 		DateTime:   time.Now(),
// 		New:        true,
// 		Identifier: uuid.NewString(),
// 		Attachment: attachment,
// 	}
// }

// func NewChatMessageWithLocation(role types.ChatMessageRole, content string, location ChatMessageLocation) ChatMessage {
// 	return ChatMessage{
// 		Role:       role,
// 		Content:    content,
// 		DateTime:   time.Now(),
// 		New:        true,
// 		Identifier: uuid.NewString(),
// 		Location:   &location,
// 	}
// }

// // this is to create a chat message from db
// func NewChatMessageFromDB(identifier string, role types.ChatMessageRole, content string, dateTime time.Time, attachment *ChatMessageAttachment) ChatMessage {
// 	return ChatMessage{
// 		Identifier: identifier,
// 		Role:       role,
// 		Content:    content,
// 		DateTime:   dateTime,
// 		Attachment: attachment,
// 	}
// }

// func (chatMessage ChatMessage) Payload() map[string]any {
// 	dic := map[string]any{
// 		"identifier": chatMessage.Identifier,
// 		"role":       chatMessage.Role,
// 		"content":    chatMessage.Content,
// 		"addedAt":  chatMessage.DateTime,
// 	}
// 	if chatMessage.Attachment != nil {
// 		dic["attachment"] = chatMessage.Attachment.Payload()
// 	}
// 	return dic
// }
