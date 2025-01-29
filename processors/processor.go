package processors

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type Processor interface {
	Init(ctx context.Context, email string, amount int64, reference string) (string, error)
	Charge(ctx context.Context, email string, amount int64, card_token string, reference string) error
	Verify(ctx context.Context, trx_id string) (bool, error)
	Webhook(ctx context.Context, r *http.Request) error
	Refund(ctx context.Context, trx_id uuid.UUID) error
}
