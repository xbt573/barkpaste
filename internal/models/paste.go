package models

import "time"

type Paste struct {
	ID           string `gorm:"primaryKey"`
	Content      []byte
	IsPersistent bool
	ExpiredAt    time.Time
}
