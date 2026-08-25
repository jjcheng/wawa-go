package dto

import "time"

type DTOBase struct {
	Id         int32     `json:"id" val:"required" example:"1"`
	EntryDate  time.Time `json:"entry_date" val:"required" example:"2025-10-01T08:08:00Z"`
	LastUpdate time.Time `json:"last_update" val:"required" example:"2025-10-01T08:08:00Z"`
}
