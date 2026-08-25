package dao_cm

import (
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/types"
)

type CachedMessage struct {
	dao.DAOBase
	ConversationIdentifier string                `gorm:"column:conversation_identifier"`
	Identifier             string                `gorm:"column:identifier"`
	Role                   types.ChatMessageRole `gorm:"column:role"`
	Content                string                `gorm:"column:content"`
	DateTime               time.Time             `gorm:"column:date_time"`
	AttachmentType         string                `gorm:"column:attachment_type"`
	AttachmentName         string                `gorm:"column:attachment_name"`
	AttachmentUrl          string                `gorm:"column:attachment_url"`
	AttachmentCaption      string                `gorm:"column:attachment_caption"`
	LocationName           string                `gorm:"column:location_name"`
	LocationLatitude       float64               `gorm:"column:location_latitude"`
	LocationLongitude      float64               `gorm:"column:location_longitude"`
	LocationAddress        string                `gorm:"column:location_address"`
	Replied                bool                  `gorm:"column:replied"`
}

func (CachedMessage) TableName() string {
	return "cm_cached_message"
}

func (cachedMessage CachedMessage) Base() dao.DAOBase {
	return cachedMessage.DAOBase
}
