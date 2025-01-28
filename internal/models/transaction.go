package models

import "github.com/google/uuid"

type TrxStatus string

var (
	TrxPending TrxStatus = "PENDING"
	TrxSuccess TrxStatus = "SUCCESS"
	TrxFailed  TrxStatus = "FAILED"
)

type Transaction struct {
	ID        uuid.UUID `json:"id"`
	InvoiceID uuid.UUID `json:"invoice_id"`
	Amount    Money     `json:"amount"`
	Status    TrxStatus `json:"status"`
}
