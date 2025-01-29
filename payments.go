package payments

import (
	"github.com/go-chi/chi/v5"
	"github.com/neghi-go/database/mongodb"
	"github.com/neghi-go/payments/billing"
	"github.com/neghi-go/payments/internal/management"
	"github.com/neghi-go/payments/internal/models"
	"github.com/neghi-go/payments/processors"
)

type Payments struct {
	url, database     string
	billing           []*billing.Billing
	payment_processor processors.Processor
}

type Option func(*Payments)

func WithDatabase(url, db string) Option {
	return func(p *Payments) {
		p.url = url
		p.database = db
	}
}

func RegisterBilling(b *billing.Billing) Option {
	return func(p *Payments) {
		p.billing = append(p.billing, b)
	}
}

func WithPaymentProcessor(pro processors.Processor) Option {
	return func(p *Payments) {
		p.payment_processor = pro
	}
}

func New(opts ...Option) *Payments {
	cfg := &Payments{
		billing: make([]*billing.Billing, 0),
	}
	cfg.billing = append(cfg.billing, management.NewManagement())

	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (p *Payments) Build() (chi.Router, error) {
	r := chi.NewRouter()
	con, err := mongodb.New(p.url, p.database)
	if err != nil {
		return nil, err
	}
	customer, err := mongodb.RegisterModel(con, "customers", models.Customer{})
	if err != nil {
		return nil, err
	}
	card, err := mongodb.RegisterModel(con, "customer_cards", models.Card{})
	if err != nil {
		return nil, err
	}
	invoice, err := mongodb.RegisterModel(con, "payment_invoices", models.Invoice{})
	if err != nil {
		return nil, err
	}
	transactions, err := mongodb.RegisterModel(con, "invoice_transactions", models.Transaction{})
	if err != nil {
		return nil, err
	}
	// register billing routes
	for _, b := range p.billing {
		route := chi.NewRouter()
		b.Init(route, &billing.BillingContext{
			Customer:     customer,
			Card:         card,
			Invoice:      invoice,
			Transactions: transactions,
			Processor:    p.payment_processor,
		})

		r.Mount("/"+b.Name, route)
	}

	return r, nil
}
