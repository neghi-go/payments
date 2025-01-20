package models

import "github.com/google/uuid"

type Invoice struct {
	ID         uuid.UUID `json:"id" db:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Amount     Money     `json:"amount"`
}
