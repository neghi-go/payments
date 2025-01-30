package models

import (
	"time"

	"github.com/google/uuid"
)

var (
	InvDraft     string = "DRAFT"
	InvIssued    string = "ISSUED"
	InvPaid      string = "PAID"
	InvExpired   string = "EXPIRED"
	InvCancelled string = "CANCELLED"
)

type Invoice struct {
	ID           uuid.UUID      `json:"id" db:"id,index,unique"`
	CustomerID   uuid.UUID      `json:"customer_id" db:"customer_id,index"`
	Amount       int64          `json:"amount" db:"amount"`
	Description  string         `json:"description" db:"description"`
	Status       string         `json:"status" db:"status"`
	LastAttempt  time.Time      `json:"last_attempt" db:"last_attempt"`
	AttemptCount int64          `json:"-" db:"attempt_count"`
	PaidAt       time.Time      `json:"paid_at" db:"paid_at"`
	ExpiresAt    time.Time      `json:"expires_at" db:"expires_at"`
	Transactions []*Transaction `json:"transactions" db:"-"`
}
