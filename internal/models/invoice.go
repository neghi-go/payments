package models

import "github.com/google/uuid"

type Invoice struct {
	ID           uuid.UUID      `json:"id" db:"id,index,unique"`
	CustomerID   uuid.UUID      `json:"customer_id" db:"customer_id,index"`
	Amount       int64          `json:"amount" db:"amount"`
	Status       string         `json:"status" db:"status"`
	Transactions []*Transaction `json:"transactions" db:"-"`
}
