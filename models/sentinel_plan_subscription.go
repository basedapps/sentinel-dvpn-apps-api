package models

import (
	"time"
)

type SentinelPlanSubscription struct {
	ID         int64     `gorm:"primary_key; not null; unique"`
	InactiveAt time.Time `gorm:"not null"`
}
