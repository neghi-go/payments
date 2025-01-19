package processors

import (
	"context"

	"github.com/google/uuid"
)

type Processor interface {
	Init(ctx context.Context, email string, amount int64)
	Charge(ctx context.Context, email string, amount int64, card_token string)
	Verify(ctx context.Context, trx_id uuid.UUID)
	Refund(ctx context.Context, trx_id uuid.UUID)
}
