package models

import "github.com/google/uuid"

type Invoice struct {
	ID     uuid.UUID `json:"id" db:"id"`
	UserID uuid.UUID
	Amount Money
}
