package models

import (
	"time"
)

type Generic struct {
	ID        uint      `gorm:"primary_key;" json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
