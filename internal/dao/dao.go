package dao

import "time"

type DAO interface {
	Base() DAOBase
	TableName() string
}

type DAOBase struct {
	Id         int32     `gorm:"column:id"`
	EntryDate  time.Time `gorm:"column:entry_date"`  //this is managed by codes in repository.Insert()
	LastUpdate time.Time `gorm:"column:last_update"` //this is managed by codes in repository.Update()
}
