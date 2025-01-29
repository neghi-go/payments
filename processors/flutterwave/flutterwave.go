package flutterwave

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/neghi-go/payments/processors"
)

type Flutterwave struct{}

// Charge implements processors.Processor.
func (f *Flutterwave) Charge(ctx context.Context, email string, amount int64, card_token string, reference string) error {
	panic("unimplemented")
}

// Init implements processors.Processor.
func (f *Flutterwave) Init(ctx context.Context, email string, amount int64, reference string) (string, error) {
	panic("unimplemented")
}

// Refund implements processors.Processor.
func (f *Flutterwave) Refund(ctx context.Context, trx_id uuid.UUID) error {
	panic("unimplemented")
}

// Verify implements processors.Processor.
func (f *Flutterwave) Verify(ctx context.Context, trx_id string) (bool, error) {
	panic("unimplemented")
}
func (f *Flutterwave) Webhook(ctx context.Context, r *http.Request) error { return nil }

type Option func(*Flutterwave)

func New(opts ...Option) *Flutterwave {
	return &Flutterwave{}
}

var _ processors.Processor = (*Flutterwave)(nil)
