package dao

import "time"

type DAO interface {
	Base() DAOBase
	TableName() string
}

type DAOBase struct {
	Id            int32     `gorm:"column:id"`
	AddedAt       time.Time `gorm:"column:added_at"`        //this is managed by codes in repository.Insert()
	LastUpdatedAt time.Time `gorm:"column:last_updated_at"` //this is managed by codes in repository.Update()
}
