package management

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neghi-go/database"
	"github.com/neghi-go/payments/billing"
	"github.com/neghi-go/payments/internal/models"
	"github.com/neghi-go/utilities"
)

type managementConfig struct{}

func NewManagement() *billing.Billing {
	return &billing.Billing{
		Name: "customers",
		Init: func(r chi.Router, ctx *billing.BillingContext) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				var (
					limit = r.URL.Query().Get("limit")
				)
				lim, _ := strconv.Atoi(limit)
				customers, err := ctx.Customer.Query(database.WithLimit(int64(lim))).All()
				if err != nil {
					utilities.JSON(w).SetStatus(utilities.ResponseError).
						SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
					return
				}
				utilities.JSON(w).SetLimit(lim).SetPage(1).SetStatusCode(http.StatusOK).
					SetStatus(utilities.ResponseSuccess).SetData(customers).Send()
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Email     string `json:"email"`
					FirstName string `json:"first_name"`
					LastName  string `json:"last_name"`
				}

				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					utilities.JSON(w).SetStatus(utilities.ResponseError).
						SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
					return
				}

				newCustomer := models.Customer{
					ID:        uuid.New(),
					Email:     body.Email,
					FirstName: body.FirstName,
					LastName:  body.LastName,
				}

				if err := ctx.Customer.Save(newCustomer); err != nil {
					utilities.JSON(w).SetStatus(utilities.ResponseError).
						SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
					return
				}
				utilities.JSON(w).SetStatusCode(http.StatusCreated).
					SetStatus(utilities.ResponseSuccess).SetData(newCustomer).Send()
			})
			r.Route("/{customer_id}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					id := r.PathValue("customer_id")
					customers, err := ctx.Customer.Query(database.WithFilter("id", uuid.MustParse(id))).First()
					if err != nil {
						utilities.JSON(w).SetStatus(utilities.ResponseError).
							SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
						return
					}
					utilities.JSON(w).SetStatusCode(http.StatusOK).
						SetStatus(utilities.ResponseSuccess).SetData(customers).Send()
				})
				r.Patch("/", func(w http.ResponseWriter, r *http.Request) {})
				r.Delete("/", func(w http.ResponseWriter, r *http.Request) {})
				r.Route("/cards", func(r chi.Router) {
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						id := r.PathValue("customer_id")
						cards, err := ctx.Card.Query(database.WithFilter("customer_id", uuid.MustParse(id))).All()
						if err != nil {
							utilities.JSON(w).SetStatus(utilities.ResponseError).
								SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
							return
						}
						utilities.JSON(w).SetStatusCode(http.StatusOK).
							SetStatus(utilities.ResponseSuccess).SetData(cards).Send()
					})
				})
				r.Route("/invoices", func(r chi.Router) {
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						id := r.PathValue("customer_id")
						invoices, err := ctx.Invoice.Query(database.WithFilter("customer_id", uuid.MustParse(id))).All()
						if err != nil {
							utilities.JSON(w).SetStatus(utilities.ResponseError).
								SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
							return
						}

						for _, inv := range invoices {
							transactions, err := ctx.Transactions.Query(database.WithFilter("invoice_id", inv.ID)).All()
							if err != nil {
								utilities.JSON(w).SetStatus(utilities.ResponseError).
									SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
								return
							}
							inv.Transactions = transactions
						}

						utilities.JSON(w).SetStatusCode(http.StatusOK).
							SetStatus(utilities.ResponseSuccess).SetData(invoices).Send()
					})
					r.Route("/{invoice_id}", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							id := r.PathValue("customer_id")
							inv_id := r.PathValue("invoice_id")
							invoice, err := ctx.Invoice.Query(
								database.WithFilter("customer_id", uuid.MustParse(id)),
								database.WithFilter("id", uuid.MustParse(inv_id)),
							).First()
							if err != nil {
								utilities.JSON(w).SetStatus(utilities.ResponseError).
									SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
								return
							}
							transactions, err := ctx.Transactions.Query(database.WithFilter("invoice_id", invoice.ID)).All()
							if err != nil {
								utilities.JSON(w).SetStatus(utilities.ResponseError).
									SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
								return
							}
							invoice.Transactions = transactions

							utilities.JSON(w).SetStatusCode(http.StatusOK).
								SetStatus(utilities.ResponseSuccess).SetData(invoice).Send()
						})
						r.Patch("/", func(w http.ResponseWriter, r *http.Request) {})
						r.Route("/transactions", func(r chi.Router) {
							r.Get("/", func(w http.ResponseWriter, r *http.Request) {
								id := r.PathValue("invoice_id")
								transactions, err := ctx.Transactions.Query(database.WithFilter("invoice_id", uuid.MustParse(id))).All()
								if err != nil {
									utilities.JSON(w).SetStatus(utilities.ResponseError).
										SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
									return
								}
								utilities.JSON(w).SetStatusCode(http.StatusOK).
									SetStatus(utilities.ResponseSuccess).SetData(transactions).Send()
							})
							r.Get("/{trx_id}", func(w http.ResponseWriter, r *http.Request) {
								id := r.PathValue("invoice_id")
								trx_id := r.PathValue("trx_id")

								transaction, err := ctx.Transactions.Query(
									database.WithFilter("invoice_id", uuid.MustParse(id)),
									database.WithFilter("id", uuid.MustParse(trx_id)),
								).First()
								if err != nil {
									utilities.JSON(w).SetStatus(utilities.ResponseError).
										SetStatusCode(http.StatusBadRequest).SetMessage(err.Error()).Send()
									return
								}
								utilities.JSON(w).SetStatusCode(http.StatusOK).
									SetStatus(utilities.ResponseSuccess).SetData(transaction).Send()
							})
						})
					})
				})
			})
		},
	}
}
