package dto_wa

import (
	"strings"

	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/types"
)

type MessageBase struct {
	AttachmentUrl     string                 `json:"attachment_url"`
	AttachmentType    types.WAAttachmentType `json:"attachment_type"`
	LocationName      string                 `json:"localtion_name"`
	LocationAddress   string                 `json:"location_address"`
	LocationLatitude  float64                `json:"location_latitude"`
	LocationLongitude float64                `json:"location_longitude"`
	HeaderText        string                 `json:"header_text"`
	BodyText          string                 `json:"body_text"`
	FooterText        string                 `json:"footer_text"`
}

func (messageBase MessageBase) Validate() []exception.InputException {
	messageBase.AttachmentUrl = strings.TrimSpace(messageBase.AttachmentUrl)
	messageBase.LocationName = strings.TrimSpace(messageBase.LocationName)
	messageBase.LocationAddress = strings.TrimSpace(messageBase.LocationAddress)
	messageBase.HeaderText = strings.TrimSpace(messageBase.HeaderText)
	messageBase.BodyText = strings.TrimSpace(messageBase.BodyText)
	messageBase.FooterText = strings.TrimSpace(messageBase.FooterText)

	inputErrors := []exception.InputException{}
	hasAttachmentUrl := messageBase.AttachmentUrl != ""
	hasAttachmentType := messageBase.AttachmentType != ""
	if hasAttachmentUrl != hasAttachmentType {
		inputErrors = append(inputErrors, exception.NewInputException("attachment_url", "attachment url and attachment type must be provided together"))
	}
	if hasAttachmentType {
		switch messageBase.AttachmentType {
		case types.WAAttachmentTypeImage, types.WAAttachmentTypeVideo, types.WAAttachmentTypeDocument:
		default:
			inputErrors = append(inputErrors, exception.NewInputException("attachment_type", "invalid attachment type"))
		}
	}
	if !hasAttachmentUrl && messageBase.HeaderText == "" {
		inputErrors = append(inputErrors, exception.NewInputException("header_text", "header text is required when there is no attachment"))
	}
	if messageBase.BodyText == "" {
		inputErrors = append(inputErrors, exception.NewInputException("body_text", "missing body text"))
	}
	return inputErrors
}
