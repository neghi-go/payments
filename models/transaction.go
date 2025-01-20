package models

import "github.com/google/uuid"

type Transaction struct {
	ID        uuid.UUID `json:"id"`
	InvoiceID uuid.UUID `json:"invoice_id"`
	Amount    Money     `json:"amount"`
	Status    Status    `json:"status"`
}
