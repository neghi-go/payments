package models

import (
	"time"
)

type Card struct {
	ID         string    `json:"-" db:"id,index,unique,required"`
	CustomerID string    `json:"-" db:"customer_id,index"`
	AuthKey    string    `json:"-" db:"auth_key"`
	Primary    bool      `json:"primary"`
	LastUsed   time.Time `json:"last_used" db:"last_used"`
}
