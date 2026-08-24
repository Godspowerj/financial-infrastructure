package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LedgerRecordInput represents the JSON payload sent from Payments -> Ledger.
type LedgerRecordInput struct {
	ReferenceID string `json:"reference_id"`
	CustomerID  string `json:"customer_id"`
	MerchantID  string `json:"merchant_id"`
	Amount      int64  `json:"amount"`
}

// LedgerClient interface allows us to swap real HTTP calls with mocks in unit tests.
type LedgerClient interface {
	RecordEntry(ctx context.Context, input LedgerRecordInput) error
}

// HTTPLedgerClient handles outbound HTTP requests to the Ledger microservice.
type HTTPLedgerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPLedgerClient creates a client configured with a base URL (e.g. "http://localhost:8082").
func NewHTTPLedgerClient(baseURL string) *HTTPLedgerClient {
	return &HTTPLedgerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// RecordEntry sends a POST /ledger/entries request to the Ledger service.
func (h *HTTPLedgerClient) RecordEntry(ctx context.Context, input LedgerRecordInput) error {
	// Construct target URL (h.baseURL + "/ledger/entries")
	url := h.baseURL + "/ledger/entries"

	// Marshal 'input' struct into JSON bytes using json.Marshal
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal ledger input: %w", err)
	}

	// Create HTTP request using http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Header: req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute request with h.httpClient.Do(req)
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status code — return error if status is not http.StatusCreated (201)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

