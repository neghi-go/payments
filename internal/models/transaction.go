package models

import "github.com/google/uuid"

var (
	TrxPending    string = "PENDING"
	TrxSuccess    string = "SUCCESS"
	TrxFailed     string = "FAILED"
	TrxAbandonned string = "ABANDONED"
)

type Transaction struct {
	ID        uuid.UUID `json:"id" db:"id,index,unique"`
	InvoiceID uuid.UUID `json:"invoice_id" db:"invoice_id,index"`
	Reference string    `json:"reference" db:"reference"`
	Status    string    `json:"status" db:"status"`
}
