package dto

import "time"

type DTOBase struct {
	Id            int32     `json:"id" val:"required" example:"1" title:"ID"`
	AddedAt       time.Time `json:"added_at" val:"required" title:"Added at" example:"2025-10-01T08:08:00Z"`
	LastUpdatedAt time.Time `json:"last_updated_at" val:"required" example:"2025-10-01T08:08:00Z"`
}
