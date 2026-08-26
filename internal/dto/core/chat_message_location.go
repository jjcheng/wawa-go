package dto_core

import (
	"fmt"
	"strings"

	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/helper"
)

type ChatMessageLocation struct {
	Name      string  `json:"name" val:"required" description:"name of the location" example:"HDB Hub"`
	Address   string  `json:"address" description:"address of the location" example:"Blk 10 Toa payph"`
	Latitude  float64 `json:"latitude" val:"required" description:"latitude of the location" example:"1.2900000"`
	Longitude float64 `json:"longitude" val:"required" description:"longitude of the location" example:"103.300000"`
}

func (chatMessageLocation *ChatMessageLocation) Validate() []exception.InputException {
	chatMessageLocation.Name = strings.TrimSpace(chatMessageLocation.Name)
	chatMessageLocation.Address = strings.TrimSpace(chatMessageLocation.Address)
	errors := []exception.InputException{}
	if chatMessageLocation.Name == "" {
		errors = append(errors, exception.NewInputException("name", "missing name"))
	}
	if chatMessageLocation.Latitude == 0 {
		errors = append(errors, exception.NewInputException("latitude", "missing latitude"))
	}
	if chatMessageLocation.Longitude == 0 {
		errors = append(errors, exception.NewInputException("longitude", "missing longitude"))
	}
	if chatMessageLocation.Address == "" {
		errors = append(errors, exception.NewInputException("address", "missing address"))
	}
	return errors
}

func (chatMessageLocation ChatMessageLocation) Payload() map[string]any {
	// name must be within 100 characters
	chatMessageLocation.Name = strings.ReplaceAll(chatMessageLocation.Name, "*", "")
	chatMessageLocation.Name = strings.ReplaceAll(chatMessageLocation.Name, ":", "")
	if len(chatMessageLocation.Name) > 80 {
		chatMessageLocation.Name = strings.Split(chatMessageLocation.Name, "\n")[0]
		if len(chatMessageLocation.Name) > 80 {
			chatMessageLocation.Name = chatMessageLocation.Name[:80]
		}
	}
	chatMessageLocation.Name = strings.TrimSpace(chatMessageLocation.Name)
	// address must be within 256 characters
	chatMessageLocation.Address = strings.ReplaceAll(chatMessageLocation.Address, "*", "")
	chatMessageLocation.Address = strings.ReplaceAll(chatMessageLocation.Address, ":", "")
	if len(chatMessageLocation.Address) > 256 {
		chatMessageLocation.Address = strings.Split(chatMessageLocation.Address, "\n")[0]
		if len(chatMessageLocation.Address) > 256 {
			chatMessageLocation.Address = chatMessageLocation.Address[:256]
		}
	}
	chatMessageLocation.Address = strings.TrimSpace(chatMessageLocation.Address)
	return map[string]any{
		"name":      chatMessageLocation.Name,
		"address":   chatMessageLocation.Address,
		"latitude":  fmt.Sprint(chatMessageLocation.Latitude),
		"longitude": fmt.Sprint(chatMessageLocation.Longitude),
	}
}

func NewChatMessageLocation(name string, address string, latitude float64, longitude float64) ChatMessageLocation {
	location := ChatMessageLocation{
		Name:      name,
		Address:   address,
		Latitude:  latitude,
		Longitude: longitude,
	}
	if helper.IsEmpty(address) {
		location.Address = "Click map to view direction"
	}
	return location
}
