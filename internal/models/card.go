package models

import (
	"time"

	"github.com/google/uuid"
)

type Card struct {
	ID         uuid.UUID `json:"-" db:"id,index,unique,required"`
	CustomerID string    `json:"-" db:"customer_id,index"`
	AuthKey    string    `json:"-" db:"auth_key"`
	LastUsed   time.Time `json:"last_used" db:"last_used"`
}
