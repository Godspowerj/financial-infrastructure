package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Common bank gateway authorization errors.
var (
	ErrInsufficientFunds = errors.New("insufficient funds in customer bank account")
	ErrBankDeclined      = errors.New("payment declined by card issuer")
	ErrBankUnreachable   = errors.New("bank simulator is offline or unreachable")
)

// BankGateway interface abstracts communication with external bank payment networks
// (e.g. Visa/Mastercard, NIBSS, or card processing networks).
type BankGateway interface {
	Authorize(ctx context.Context, payment *Payment) (bool, string, error)
}

// MockBankGateway is an in-memory simulator implementation of BankGateway for unit tests.
type MockBankGateway struct{}

// NewMockBankGateway creates a new MockBankGateway instance.
func NewMockBankGateway() *MockBankGateway {
	return &MockBankGateway{}
}

// Authorize simulates in-memory authorization.
func (g *MockBankGateway) Authorize(ctx context.Context, payment *Payment) (bool, string, error) {
	if payment.Amount%100 == 99 {
		return false, "insufficient_funds", ErrInsufficientFunds
	}
	return true, "", nil
}

// HTTPBankGateway connects to a standalone Bank Simulator microservice over HTTP.
type HTTPBankGateway struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPBankGateway creates a new HTTPBankGateway pointing at the given Bank Simulator URL.
func NewHTTPBankGateway(baseURL string) *HTTPBankGateway {
	return &HTTPBankGateway{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second, // 3 second timeout for HTTP requests
		},
	}
}

type bankAuthorizeReq struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

type bankAuthorizeResp struct {
	Approved  bool   `json:"approved"`
	Reason    string `json:"reason,omitempty"`
	Reference string `json:"reference,omitempty"`
}

// Authorize makes an HTTP POST request to the Bank Simulator with retries and timeout context.
func (h *HTTPBankGateway) Authorize(ctx context.Context, payment *Payment) (bool, string, error) {
	url := h.baseURL + "/authorize"

	payload := bankAuthorizeReq{
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return false, "", fmt.Errorf("http gateway: failed marshaling request: %w", err)
	}

	maxRetries := 3
	backoff := 100 * time.Millisecond

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewBuffer(body))
		if err != nil {
			cancel()
			return false, "", fmt.Errorf("http gateway: failed creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.httpClient.Do(req)
		cancel()

		if err == nil && resp.StatusCode == http.StatusOK {
			var bankResp bankAuthorizeResp
			if decodeErr := json.NewDecoder(resp.Body).Decode(&bankResp); decodeErr != nil {
				resp.Body.Close()
				return false, "", fmt.Errorf("http gateway: failed decoding bank response: %w", decodeErr)
			}
			resp.Body.Close()

			return bankResp.Approved, bankResp.Reason, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("bank responded with status %d", resp.StatusCode)
		}

		time.Sleep(backoff)
		backoff *= 2 // Exponential backoff: 100ms, 200ms, 400ms
	}

	return false, "bank_unreachable", fmt.Errorf("http gateway: %w (after %d attempts): %v", ErrBankUnreachable, maxRetries, lastErr)
}
