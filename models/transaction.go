package models

import "github.com/google/uuid"

type Transaction struct {
	ID        uuid.UUID
	InvoiceID uuid.UUID
	Amount    Money
	Status    Status
}
