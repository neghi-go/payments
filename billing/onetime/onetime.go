package onetime

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neghi-go/database"
	"github.com/neghi-go/payments/billing"
	"github.com/neghi-go/payments/internal/models"
	"github.com/neghi-go/payments/utils"
)

type Action string

var (
	create Action = "create"
)

type Options func(*depositBilling) error

type depositBilling struct {
	notify func() error
}

func NewDepositBilling(opts ...Options) *billing.Billing {
	_ = &depositBilling{}
	return &billing.Billing{
		Name: "onetime",
		Init: func(r chi.Router, ctx *billing.BillingContext) {
			r.Post("/charge", func(w http.ResponseWriter, r *http.Request) {
				var (
					amount  int64
					invoice *models.Invoice
					trx     *models.Transaction
				)
				action := r.URL.Query().Get("action")
				//get payment data
				var body struct {
					CustomerID string `json:"customer_id"`
					Amount     int64  `json:"amount"`
					InvoiceID  string `json:"invoice_id"`
				}

				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				customer, err := ctx.Customer.Query(database.WithFilter("id", uuid.MustParse(body.CustomerID))).First()
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				switch Action(action) {
				case create:
					//create a new invoice
					invoice = &models.Invoice{
						ID:         uuid.New(),
						CustomerID: uuid.MustParse(body.CustomerID),
						Amount:     int64(body.Amount),
						Status:     "PENDING",
					}
					if err := ctx.Invoice.Save(*invoice); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					//create a new transaction
					trx = &models.Transaction{
						ID:        uuid.New(),
						InvoiceID: invoice.ID,
						Amount:    invoice.Amount,
						Status:    models.TrxPending,
						Reference: utils.GenerateReference(12),
					}
					if err := ctx.Transactions.Save(*trx); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					amount = int64(trx.Amount)
				default:
					//check for invoice status
					//if status is not paid, canceled, proceed
					invoice, err = ctx.Invoice.Query(database.WithFilter("id", body.InvoiceID)).First()
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
					tranx, err := ctx.Transactions.Query(
						database.WithFilter("invoice_id", invoice.ID),
					).All()
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					trx = tranx[len(tranx)-1]
					if trx.Status == models.TrxFailed {
						//create new trx
						trx = &models.Transaction{}
						if err := ctx.Transactions.Save(*trx); err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						amount = int64(invoice.Amount)
					}
					if trx.Status == models.TrxPending {
						var valid bool
						//verify transaction and update accordingly
						if valid, err = ctx.Processor.Verify(r.Context(), trx.ID.String()); err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}

						if valid {
							invoice.Status = ""
							trx.Status = models.TrxSuccess
							if err := ctx.Invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							if err := ctx.Transactions.Query(database.WithFilter("id", trx)).Update(*trx); err != nil {
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							w.WriteHeader(http.StatusOK)
							return
						}
					}
					if trx.Status == models.TrxSuccess {
						//update invoice to paid.
						invoice.Status = ""
						if err := ctx.Invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						w.Write([]byte("invoice has been paid."))
						w.WriteHeader(http.StatusOK)
						return
					}
				}
				//check if user has a valid card, if yes, attempt to charge card else, generate payment url and redirect
				validCard, err := ctx.Card.Query(database.WithFilter("customer_id", uuid.MustParse(body.CustomerID))).First()
				if err != nil {
					auth_url, err := ctx.Processor.Init(r.Context(), customer.Email, amount, trx.Reference)
					if err != nil {
						w.WriteHeader(http.StatusOK)
						return
					}
					fmt.Println(auth_url)
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(auth_url))
					return
				}
				if err := ctx.Processor.Charge(r.Context(), customer.Email, amount, validCard.AuthKey, trx.Reference); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
			})
			r.Post("/verify/{id}", func(w http.ResponseWriter, r *http.Request) {
				var valid bool
				var err error
				TransactionID := r.PathValue("id")

				trx, err := ctx.Transactions.Query(database.WithFilter("id", uuid.MustParse(TransactionID))).First()
				if err != nil {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				if valid, err = ctx.Processor.Verify(r.Context(), trx.Reference); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(err.Error()))
					return
				}
				if valid {

					//if successfull, look and update invoice status
					trx.Status = models.TrxSuccess

					if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					inv, err := ctx.Invoice.Query(database.WithFilter("id", trx.InvoiceID)).First()
					if err != nil {
						w.WriteHeader(http.StatusBadGateway)
						return
					}
					inv.Status = "PAID"
					if err := ctx.Invoice.Query(database.WithFilter("id", inv.ID)).Update(*inv); err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}

				w.WriteHeader(http.StatusOK)
			})
		},
	}
}
