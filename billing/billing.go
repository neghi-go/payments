package billing

import (
	"github.com/go-chi/chi/v5"
	"github.com/neghi-go/database"
	"github.com/neghi-go/payments/internal/models"
	"github.com/neghi-go/payments/processors"
)

type BillingContext struct {
	Customer     database.Model[models.Customer]
	Card         database.Model[models.Card]
	Invoice      database.Model[models.Invoice]
	Transactions database.Model[models.Transaction]
	Processor    processors.Processor
}

type Billing struct {
	Name string
	Init func(r chi.Router, ctx BillingContext)
}
