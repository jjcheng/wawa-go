package dao_cm

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Message struct {
	dao.DAOBase
	Identifier        string                `gorm:"column:identifier"` // to track status
	ConversationId    int32                 `gorm:"column:conversation_id"`
	SessionId         string                `gorm:"column:session_id"`
	Role              types.ChatMessageRole `gorm:"column:role"`
	Content           string                `gorm:"column:content"`
	DateTime          time.Time             `gorm:"column:date_time"`
	AttachmentType    string                `gorm:"column:attachment_type"`
	AttachmentName    string                `gorm:"column:attachment_name"`
	AttachmentUrl     string                `gorm:"column:attachment_url"`
	AttachmentCaption string                `gorm:"column:attachment_caption"`
	LocationName      string                `gorm:"column:location_name"`
	LocationLatitude  float64               `gorm:"column:location_latitude"`
	LocationLongitude float64               `gorm:"column:location_longitude"`
	LocationAddress   string                `gorm:"column:location_address"`
	ParentIdentifier  string                `gorm:"column:parent_identifier"`
}

func (Message) TableName() string {
	return "cm_message"
}

func (chatMessage Message) Base() dao.DAOBase {
	return chatMessage.DAOBase
}
