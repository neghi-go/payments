package paystack

import (
	"context"

	"github.com/google/uuid"
	"github.com/neghi-go/payments/processors"
)

type Paystack struct{}

type Option func(*Paystack)

// Charge implements processors.Processor.
func (p *Paystack) Charge(ctx context.Context, email string, amount int64, card_token string) {
	panic("unimplemented")
}

// Init implements processors.Processor.
func (p *Paystack) Init(ctx context.Context, email string, amount int64) {
	panic("unimplemented")
}

// Refund implements processors.Processor.
func (p *Paystack) Refund(ctx context.Context, trx_id uuid.UUID) {
	panic("unimplemented")
}

// Verify implements processors.Processor.
func (p *Paystack) Verify(ctx context.Context, trx_id uuid.UUID) {
	panic("unimplemented")
}

func New(opts ...Option) *Paystack {
	return &Paystack{}
}

var _ processors.Processor = (*Paystack)(nil)
