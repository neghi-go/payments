package payments

import (
	"github.com/go-chi/chi/v5"
	"github.com/neghi-go/database"
	"github.com/neghi-go/database/mongodb"
	"github.com/neghi-go/payments/billing"
	"github.com/neghi-go/payments/internal/management"
	"github.com/neghi-go/payments/internal/models"
)

type Payments struct {
	billing []*billing.Billing
}

type Option func(*Payments)

func New(opts ...Option) *Payments {
	cfg := &Payments{
		billing: make([]*billing.Billing, 0),
	}
	cfg.billing = append(cfg.billing, management.NewManagement())

	return cfg
}

func (p *Payments) Build() (chi.Router, error) {
	r := chi.NewRouter()

	//register main/ management routes

	// register billing routes
	for _, b := range p.billing {
		route := chi.NewRouter()
		b.Init(route, billing.BillingContext{
			Customer: handleConnection("mongodb", "", "", "customers", models.Customer{}),
		})

		r.Mount("/"+b.Name, route)
	}

	return r, nil
}

func handleConnection[T any](engine string, url, database string, coll string, model T) database.Model[T] {
	switch engine {
	case "mongodb":
		con, err := mongodb.New(url, database)
		if err != nil {
			panic(err)
		}
		db, err := mongodb.RegisterModel(con, coll, model)
		if err != nil {
			panic(err)
		}

		return db
	default:
		return nil
	}
}
