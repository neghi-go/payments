package management

import (
	"github.com/go-chi/chi/v5"
	"github.com/neghi-go/payments/billing"
)

type managementConfig struct{}

func NewManagement() *billing.Billing {
	return &billing.Billing{
		Name: "management",
		Init: func(r chi.Router, ctx billing.BillingContext) {},
	}
}
