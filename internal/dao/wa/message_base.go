package dao_wa

import "github.com/jjcheng/wawa-go/internal/types"

type MessageBase struct {
	AttachmentUrl     string                 `gorm:"column:attachment_url"`
	AttachmentType    types.WAAttachmentType `gorm:"column:attachment_type"`
	LocationName      string                 `gorm:"column:localtion_name"`
	LocationAddress   string                 `gorm:"column:location_address"`
	LocationLatitude  float64                `gorm:"column:location_latitude"`
	LocationLongitude float64                `gorm:"column:location_longitude"`
	HeaderText        string                 `gorm:"column:header_text"`
	BodyText          string                 `gorm:"column:body_text"`
	FooterText        string                 `gorm:"column:footer_text"`
}
