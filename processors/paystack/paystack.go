package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/neghi-go/payments/processors"
)

var (
	base_url       = "https://api.paystack.co"
	initialize_url = "/transaction/initialize"
	verify_url     = "/transaction/verify"
	charge_url     = ""
	refund_url     = ""
)

type Paystack struct {
	key string
}

type Option func(*Paystack)

type initiateResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AuthorizationURL string `json:"authorization_url"`
		AccessCode       string `json:"access_code"`
		Reference        string `json:"refernce"`
	}
}

type trxResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Amount          int64                  `json:"amount"`
		Currency        string                 `json:"currency"`
		TransactionDate time.Time              `json:"transaction_date"`
		Status          string                 `json:"status"`
		Reference       string                 `json:"reference"`
		Domain          string                 `json:"domain"`
		Metadata        string                 `json:"metadata"`
		GateWayResponse string                 `json:"gateway_response"`
		Message         string                 `json:"message"`
		Channel         string                 `json:"channel"`
		IPAddress       string                 `json:"ip_address"`
		Log             map[string]interface{} `json:"log"`
		Fees            int64                  `json:"fees"`
		Authorization   struct {
			AuthorizationCode string `json:"authorization_code"`
			Bin               string `json:"bin"`
			Last4             string `json:"last4"`
			ExpMonth          string `json:"exp_month"`
			ExpYear           string `json:"exp_year"`
			Channel           string `json:"channel"`
			CardType          string `json:"card_type"`
			Bank              string `json:"bank"`
			CountryCode       string `json:"country_code"`
			Brand             string `json:"brand"`
			Reuseable         string `json:"reuseable"`
			Signature         string `json:"signature"`
			AccountName       string `json:"account_name"`
		} `json:"authorization"`
		Customer struct {
			ID           int    `json:"id"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			Email        string `json:"email"`
			CustomerCode string `json:"customer_code"`
			Phone        string `json:"phone"`
			Metadata     struct {
				CustomFields []struct {
					DisplayName  string `json:"display_name"`
					VariableName string `json:"variable_name"`
					Value        string `json:"value"`
				} `json:"custom_fields"`
			} `json:"metadata"`
			RiskAction               string `json:"risk_action"`
			InternationalPhoneFormat string `json:"international_format_phone"`
		}
		ID int `json:"id"`
	}
}

type refundResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Transaction struct {
			ID            int       `json:"id"`
			Reference     string    `json:"reference"`
			Domain        string    `json:"domain"`
			Amount        int64     `json:"amount"`
			PaidAt        time.Time `json:"paid_at"`
			Channel       string    `json:"channel"`
			Currency      string    `json:"currency"`
			Authorization struct {
				AuthorizationCode string `json:"authorization_code"`
				Bin               string `json:"bin"`
				Last4             string `json:"last4"`
				ExpMonth          string `json:"exp_month"`
				ExpYear           string `json:"exp_year"`
				Channel           string `json:"channel"`
				CardType          string `json:"card_type"`
				Bank              string `json:"bank"`
				CountryCode       string `json:"country_code"`
				Brand             string `json:"brand"`
				Reuseable         string `json:"reuseable"`
				Signature         string `json:"signature"`
				AccountName       string `json:"account_name"`
			} `json:"authorization"`
		}
		ID             int       `json:"id"`
		Integration    int       `json:"integration"`
		DeductedAmount int64     `json:"deducted_amount"`
		MerchantNote   string    `json:"merchant_note"`
		CustomerNote   string    `json:"customer_note"`
		RefundedBy     string    `json:"refunded_by"`
		ExpectedAt     time.Time `json:"expected_at"`
		FullyDeducted  bool      `json:"fully_deducted"`
		Amount         int64     `json:"amount"`
		Currency       string    `json:"currency"`
		Status         string    `json:"status"`
		Domain         string    `json:"domain"`
		Channel        string    `json:"channel"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	}
}

// Charge implements processors.Processor.
func (p *Paystack) Charge(ctx context.Context, email string, amount int64, card_token string, reference string) error {
	client := http.DefaultClient
	var res_body trxResponse
	buf := &bytes.Buffer{}
	body := struct {
		Card_Token string
		Email      string `json:"email"`
		Amount     int64  `json:"amount"`
		Reference  string `json:"reference"`
	}{
		Card_Token: card_token,
		Email:      email,
		Amount:     amount,
		Reference:  reference,
	}
	err := json.NewEncoder(buf).Encode(body)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base_url+charge_url, buf)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+p.key)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&res_body); err != nil {
		return err
	}
	return nil
}

// Init implements processors.Processor.
func (p *Paystack) Init(ctx context.Context, email string, amount int64, reference string) (string, error) {
	var res_body initiateResponse

	buf := &bytes.Buffer{}

	body := struct {
		Email     string `json:"email"`
		Amount    int64  `json:"amount"`
		Reference string `json:"reference"`
	}{
		Email:     email,
		Amount:    amount,
		Reference: reference,
	}

	err := json.NewEncoder(buf).Encode(body)
	if err != nil {
		return "", err
	}
	client := http.DefaultClient

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base_url+initialize_url, buf)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+p.key)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&res_body); err != nil {
		return "", err
	}
	fmt.Println(res_body)

	return res_body.Data.AuthorizationURL, nil
}

// Refund implements processors.Processor.
func (p *Paystack) Refund(ctx context.Context, trx_id uuid.UUID) error {
	client := http.DefaultClient
	var res_body refundResponse

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base_url+refund_url+"/"+trx_id.String(), nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+p.key)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&res_body); err != nil {
		return err
	}
	return nil
}

// Verify implements processors.Processor.
func (p *Paystack) Verify(ctx context.Context, trx_id string) (bool, error) {
	url := &bytes.Buffer{}

	url.WriteString(base_url)
	url.WriteString(verify_url)
	client := http.DefaultClient
	var res_body trxResponse

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url.String()+"/"+trx_id, nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+p.key)

	res, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(&res_body); err != nil {
		return false, err
	}
	return true, nil
}

func (p *Paystack) Webhook(ctx context.Context, r *http.Request) error { return nil }

func SetKey(key string) Option {
	return func(p *Paystack) {
		p.key = key
	}
}

func New(opts ...Option) *Paystack {
	cfg := &Paystack{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

var _ processors.Processor = (*Paystack)(nil)
