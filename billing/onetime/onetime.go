package onetime

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/neghi-go/database"
	"github.com/neghi-go/payments/billing"
	"github.com/neghi-go/payments/internal/models"
	"github.com/neghi-go/payments/processors"
	"github.com/neghi-go/payments/utils"
	"github.com/neghi-go/utilities"
)

type Action string

var (
	initialize Action = "init"
)

type Options func(*depositBilling) error

type depositBilling struct {
	reference_length int
	notify           func() error
}

func NewDepositBilling(opts ...Options) *billing.Billing {
	cfg := &depositBilling{
		reference_length: 12,
	}
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
					utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
						SetStatusCode(http.StatusBadRequest).Send()
					return
				}

				customer, err := ctx.Customer.Query(database.WithFilter("id", uuid.MustParse(body.CustomerID))).First()
				if err != nil {
					utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
						SetStatusCode(http.StatusNotFound).Send()
					return
				}

				switch Action(action) {
				case initialize:
					//create a new invoice
					invoice = &models.Invoice{
						ID:           uuid.New(),
						CustomerID:   uuid.MustParse(body.CustomerID),
						Description:  "description",
						Amount:       body.Amount,
						Status:       models.InvIssued,
						AttemptCount: 1,
						PaidAt:       time.Time{},
						LastAttempt:  time.Now().UTC(),
						ExpiresAt:    time.Now().Add(time.Hour * 24).UTC(),
					}
					if err := ctx.Invoice.Save(*invoice); err != nil {
						utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
							SetStatusCode(http.StatusInternalServerError).Send()
						return
					}

					//create a new transaction
					trx = &models.Transaction{
						ID:        uuid.New(),
						InvoiceID: invoice.ID,
						Status:    models.TrxPending,
						Reference: utils.GenerateReference(cfg.reference_length),
					}
					if err := ctx.Transactions.Save(*trx); err != nil {
						utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
							SetStatusCode(http.StatusInternalServerError).Send()
						return
					}
					amount = invoice.Amount
				default:
					//check for invoice status
					//if status is not paid, canceled, proceed
					invoice, err = ctx.Invoice.Query(database.WithFilter("id", uuid.MustParse(body.InvoiceID))).First()
					if err != nil {
						utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
							SetStatusCode(http.StatusNotFound).Send()
						return
					}

					if invoice.Status == models.InvPaid {
						utilities.JSON(w).SetMessage("Invoice has been Cleared Already!").SetStatus(utilities.ResponseSuccess).
							SetStatusCode(http.StatusOK).Send()
						return
					}
					if invoice.Status == models.InvExpired || invoice.Status == models.InvCancelled {
						utilities.JSON(w).SetMessage("This invoice is no longer valid!").SetStatus(utilities.ResponseFail).
							SetStatusCode(http.StatusBadRequest).Send()
						return
					}

					if invoice.Status == models.InvIssued {
						if time.Now().Unix() > invoice.ExpiresAt.Unix() {
							invoice.Status = models.InvExpired
							if err := ctx.Invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusNotFound).Send()
								return
							}
							utilities.JSON(w).SetMessage("Invoice is Expired, please try again").SetStatus(utilities.ResponseFail).
								SetStatusCode(http.StatusBadRequest).Send()
							return
						}
						//check state of last transaction
						//if status is not pending or success, proceed
						tranx, err := ctx.Transactions.Query(
							database.WithFilter("invoice_id", invoice.ID),
						).All()
						if err != nil {
							utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
								SetStatusCode(http.StatusBadRequest).Send()
							return
						}

						trx = tranx[len(tranx)-1]
						if trx.Status == models.TrxFailed {
							invoice.LastAttempt = time.Now().UTC()
							invoice.AttemptCount += 1
							//create new trx
							trx = &models.Transaction{
								ID:        uuid.New(),
								InvoiceID: invoice.ID,
								Status:    models.TrxPending,
								Reference: utilities.Generate(cfg.reference_length),
							}
							if err := ctx.Transactions.Save(*trx); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusBadRequest).Send()
								return
							}
							amount = invoice.Amount
						}
						if trx.Status == models.TrxPending {
							var res processors.VerifyState
							//verify transaction and update accordingly
							if res, err = ctx.Processor.Verify(r.Context(), trx.Reference); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
									SetStatusCode(http.StatusInternalServerError).Send()
								return
							}

							if res == processors.Success {
								invoice.Status = models.InvPaid
								invoice.PaidAt = time.Now().UTC()
								trx.Status = models.TrxSuccess
								if err := ctx.Invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
									utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
										SetStatusCode(http.StatusBadRequest).Send()
									return
								}
								if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
									utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
										SetStatusCode(http.StatusBadRequest).Send()
									return
								}
								utilities.JSON(w).SetMessage("Invoice has been cleared!").SetStatus(utilities.ResponseSuccess).
									SetStatusCode(http.StatusOK).Send()
								return
							}
							if res == processors.Failed {
								trx.Status = models.TrxFailed
								if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
									utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
										SetStatusCode(http.StatusBadRequest).Send()
									return
								}
								utilities.JSON(w).SetMessage("Your Transaction Failed, Please try again").SetStatus(utilities.ResponseSuccess).
									SetStatusCode(http.StatusOK).Send()
								return
							}
							if res == processors.Abandoned {
								trx.Status = models.TrxAbandonned
								if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
									utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
										SetStatusCode(http.StatusBadRequest).Send()
									return
								}
								utilities.JSON(w).SetMessage("Your Transaction Could not be completed, Please try again").
									SetStatus(utilities.ResponseSuccess).
									SetStatusCode(http.StatusOK).Send()
								return
							}
							if res == processors.Pending {
								trx.Status = models.TrxPending
								if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
									utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
										SetStatusCode(http.StatusBadRequest).Send()
									return
								}
								utilities.JSON(w).SetMessage("Your Transaction is still pending, please try again later").
									SetStatus(utilities.ResponseSuccess).
									SetStatusCode(http.StatusOK).Send()
								return
							}
						}
						if trx.Status == models.TrxSuccess {
							//update invoice to paid.
							utilities.JSON(w).SetMessage("Invoice has been cleared!").SetStatus(utilities.ResponseSuccess).
								SetStatusCode(http.StatusOK).Send()
							return
						}
					}
					if invoice.Status == models.InvDraft {
						utilities.JSON(w).SetMessage("Invoice is not supported on this resource").SetStatus(utilities.ResponseFail).
							SetStatusCode(http.StatusBadRequest).Send()
						return
					}

				}
				//check if user has a valid card, if yes, attempt to charge card else, generate payment url and redirect
				validCard, err := ctx.Card.Query(database.WithFilter("customer_id", uuid.MustParse(body.CustomerID))).First()
				if err != nil {
					auth_url, err := ctx.Processor.Init(r.Context(), customer.Email, amount, trx.Reference)
					if err != nil {
						utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
							SetStatusCode(http.StatusInternalServerError).Send()
						return
					}
					utilities.JSON(w).SetStatus(utilities.ResponseSuccess).
						SetStatusCode(http.StatusOK).SetMessage(auth_url).Send()
					return
				}
				if err := ctx.Processor.Charge(r.Context(), customer.Email, amount, validCard.AuthKey, trx.Reference); err != nil {
					utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
						SetStatusCode(http.StatusInternalServerError).Send()
					return
				}
				utilities.JSON(w).SetMessage("Charge Successfull").SetStatus(utilities.ResponseSuccess).
					SetStatusCode(http.StatusBadRequest).Send()
			})
			r.Post("/verify/{id}", func(w http.ResponseWriter, r *http.Request) {
				var err error
				InvoiceID := r.PathValue("id")

				invoice, err := ctx.Invoice.Query(database.WithFilter("id", uuid.MustParse(InvoiceID))).First()
				if err != nil {
					utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
						SetStatusCode(http.StatusNotFound).Send()
					return
				}

				if invoice.Status == models.InvPaid {
					utilities.JSON(w).SetMessage("Invoice has been Cleared Already!").SetStatus(utilities.ResponseSuccess).
						SetStatusCode(http.StatusOK).Send()
					return
				}
				if invoice.Status == models.InvExpired || invoice.Status == models.InvCancelled {
					utilities.JSON(w).SetMessage("This invoice is no longer valid!").SetStatus(utilities.ResponseFail).
						SetStatusCode(http.StatusBadRequest).Send()
					return
				}

				if invoice.Status == models.InvIssued {
					if time.Now().Unix() > invoice.ExpiresAt.Unix() {
						invoice.Status = models.InvExpired
						if err := ctx.Invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
							utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
								SetStatusCode(http.StatusNotFound).Send()
							return
						}
						utilities.JSON(w).SetMessage("Invoice is Expired, please try again").SetStatus(utilities.ResponseFail).
							SetStatusCode(http.StatusBadRequest).Send()
						return
					}
					//check state of last transaction
					//if status is not pending or success, proceed
					tranx, err := ctx.Transactions.Query(
						database.WithFilter("invoice_id", invoice.ID),
					).All()
					if err != nil {
						utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
							SetStatusCode(http.StatusBadRequest).Send()
						return
					}

					trx := tranx[len(tranx)-1]
					if trx.Status == models.TrxFailed {
						utilities.JSON(w).SetMessage("Transaction Failed, please try again").SetStatus(utilities.ResponseError).
							SetStatusCode(http.StatusInternalServerError).Send()
						return
					}
					if trx.Status == models.TrxPending {
						var res processors.VerifyState
						//verify transaction and update accordingly
						if res, err = ctx.Processor.Verify(r.Context(), trx.Reference); err != nil {
							utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseError).
								SetStatusCode(http.StatusInternalServerError).Send()
							return
						}

						if res == processors.Success {
							invoice.Status = models.InvPaid
                            invoice.PaidAt = time.Now().UTC()
							trx.Status = models.TrxSuccess
							if err := ctx.Invoice.Query(database.WithFilter("id", invoice.ID)).Update(*invoice); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusBadRequest).Send()
								return
							}
							if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusBadRequest).Send()
								return
							}
							utilities.JSON(w).SetMessage("Invoice has been cleared!").SetStatus(utilities.ResponseSuccess).
								SetStatusCode(http.StatusOK).Send()
							return
						}
						if res == processors.Failed {
							trx.Status = models.TrxFailed
							if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusBadRequest).Send()
								return
							}
							utilities.JSON(w).SetMessage("Your Transaction Failed, Please try again").SetStatus(utilities.ResponseSuccess).
								SetStatusCode(http.StatusOK).Send()
							return
						}
						if res == processors.Abandoned {
							trx.Status = models.TrxAbandonned
							if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusBadRequest).Send()
								return
							}
							utilities.JSON(w).SetMessage("Your Transaction Could not be completed, Please try again").
								SetStatus(utilities.ResponseSuccess).
								SetStatusCode(http.StatusOK).Send()
							return
						}
						if res == processors.Pending {
							trx.Status = models.TrxPending
							if err := ctx.Transactions.Query(database.WithFilter("id", trx.ID)).Update(*trx); err != nil {
								utilities.JSON(w).SetMessage(err.Error()).SetStatus(utilities.ResponseFail).
									SetStatusCode(http.StatusBadRequest).Send()
								return
							}
							utilities.JSON(w).SetMessage("Your Transaction is still pending, please try again later").
								SetStatus(utilities.ResponseSuccess).
								SetStatusCode(http.StatusOK).Send()
							return
						}
					}
					if trx.Status == models.TrxSuccess {
						utilities.JSON(w).SetMessage("Invoice has been cleared!").SetStatus(utilities.ResponseSuccess).
							SetStatusCode(http.StatusOK).Send()
						return
					}
				}
				if invoice.Status == models.InvDraft {
					utilities.JSON(w).SetMessage("Invoice is not supported on this resource").SetStatus(utilities.ResponseFail).
						SetStatusCode(http.StatusBadRequest).Send()
					return
				}

			})
		},
	}
}
