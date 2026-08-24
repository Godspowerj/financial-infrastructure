package transfers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type LedgerClient interface {
	GetBalance(ctx context.Context, accountID string) (int64, error)
	PostTransferEntry(ctx context.Context, referenceID string, senderID string, receiverID string, amount int64) error
}

type HTTPClient struct {
	http.Client
	baseURL string
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		Client:  http.Client{Timeout: 10 * time.Second},
		baseURL: baseURL,
	}
}

func (c *HTTPClient) GetBalance(ctx context.Context, accountID string) (int64, error) {
	url := c.baseURL + "/accounts/" + accountID + "/balance"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create get balance request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute get balance request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ledger service returned status %d", resp.StatusCode)
	}

	var balance struct {
		Balance int64 `json:"balance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
		return 0, fmt.Errorf("failed to decode ledger response: %w", err)
	}

	return balance.Balance, nil
}

func (c *HTTPClient) PostTransferEntry(ctx context.Context, referenceID string, senderID string, receiverID string, amount int64) error {
	url := c.baseURL + "/ledger/entries"

	payload := map[string]any{
		"reference_id": referenceID,
		"customer_id":  senderID,
		"merchant_id":  receiverID,
		"amount":       amount,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal ledger transfer payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create post transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute post transfer request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ledger service returned unexpected status %d", resp.StatusCode)
	}

	return nil
}
