package onetime

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/neghi-go/database"
	"github.com/neghi-go/payments/billing"
	"github.com/neghi-go/payments/internal/models"
)

type Action string

var (
	create Action = "create"
)

type Options func(*depositBilling) error

type depositBilling struct {
	customer    database.Model[models.Customer]
	invoice     database.Model[models.Invoice]
	card        database.Model[models.Card]
	transaction database.Model[models.Transaction]
	notify      func() error
}

func NewDepositBilling(opts ...Options) *billing.Billing {
	cfg := &depositBilling{}
	return &billing.Billing{
		Name: "deposit",
		Init: func(r chi.Router, ctx billing.BillingContext) {
			r.Post("/charge", func(w http.ResponseWriter, r *http.Request) {
				var amount int64
				action := r.URL.Query().Get("action")
				//get payment data
				var body struct {
					CustomerID string `json:"customer_id"`
					Amount     int    `json:"amount"`
					InvoiceID  string `json:"invoice_id"`
				}

				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				customer, err := cfg.customer.Query(database.WithFilter("id", body.CustomerID)).First()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				switch Action(action) {
				case create:
					//create a new invoice
					invoice := &models.Invoice{}
					if err := cfg.invoice.Save(*invoice); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					//create a new transaction
					trx := &models.Transaction{}
					if err := cfg.transaction.Save(*trx); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					amount = int64(trx.Amount)
				default:
					//check for invoice status
					//if status is not paid, canceled, proceed
					invoice, err := cfg.invoice.Query(database.WithFilter("id", body.InvoiceID)).First()
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					if invoice.Status != "PAID" && invoice.Status != "CANCELED" {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					//check state of last transaction
					//if status is not pending or success, proceed
					trx, err := cfg.transaction.Query(
						database.WithFilter("invoice_id", invoice.ID),
					).All()
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					lastTRX := trx[len(trx)-1]
					if lastTRX.Status == models.TrxFailed {
						//create new trx
						trx := models.Transaction{}
						if err := cfg.transaction.Save(trx); err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						amount = int64(invoice.Amount)
					}
					if lastTRX.Status == models.TrxPending {
						var valid bool
						//verify transaction and update accordingly
						if valid, err = ctx.Processor.Verify(r.Context(), lastTRX.ID.String()); err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}

						if valid {
							invoice.Status = ""
							lastTRX.Status = models.TrxSuccess
							if err := cfg.invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							if err := cfg.transaction.Query(database.WithFilter("id", lastTRX)).Update(*lastTRX); err != nil {
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							w.WriteHeader(http.StatusOK)
							return
						}
					}
					if lastTRX.Status == models.TrxSuccess {
						//update invoice to paid.
						invoice.Status = ""
						if err := cfg.invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						w.Write([]byte("invoice has been paid."))
						w.WriteHeader(http.StatusOK)
						return
					}
				}
				//check if user has a valid card, if yes, attempt to charge card else, generate payment url and redirect
				validCard, err := cfg.card.Query(database.WithFilter("", "")).First()
				if err != nil {
					if err := ctx.Processor.Charge(r.Context(), customer.Email, amount, validCard.AuthKey); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusOK)
					return
				}
				auth_url, err := ctx.Processor.Init(r.Context(), customer.Email, amount)
				if err != nil {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.Write([]byte(auth_url))
				w.WriteHeader(http.StatusOK)
				return
			})
			r.Post("/verify", func(w http.ResponseWriter, r *http.Request) {
				var valid bool
				var err error
				var body struct {
					TransactionID string `json:"transaction_id"`
				}
				if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if valid, err = ctx.Processor.Verify(r.Context(), body.TransactionID); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if valid {

					//if successfull, look and update invoice status
					trx, err := cfg.transaction.Query(database.WithFilter("id", body.TransactionID)).First()
					if err != nil {
						w.WriteHeader(http.StatusBadGateway)
						return
					}

					trx.Status = "PAID"

					if err := cfg.transaction.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					inv, err := cfg.invoice.Query(database.WithFilter("id", trx.InvoiceID)).First()
					if err != nil {
						w.WriteHeader(http.StatusBadGateway)
						return
					}
					inv.Status = "PAID"
					if err := cfg.invoice.Query(database.WithFilter("id", inv.ID)).Update(*inv); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}

				w.WriteHeader(http.StatusOK)
			})
		},
	}
}
