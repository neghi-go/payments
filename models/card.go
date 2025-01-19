package models

import (
	"time"

	"github.com/google/uuid"
)

type Card struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	AuthKey  string `json:"-" db:"auth_key"`
	LastUsed time.Time
}
