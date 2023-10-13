package models

import (
	"time"
)

type SentinelNodeSubscription struct {
	ID          int64     `gorm:"primary_key; not null; unique"`
	NodeAddress string    `gorm:"not null"`
	InactiveAt  time.Time `gorm:"not null"`
}
