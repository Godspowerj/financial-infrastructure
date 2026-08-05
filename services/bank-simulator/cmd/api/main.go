package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// AuthorizeRequest is sent by the Payments service to request authorization.
type AuthorizeRequest struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

// AuthorizeResponse is returned by the Bank Simulator to the Payments service.
type AuthorizeResponse struct {
	Approved  bool   `json:"approved"`
	Reason    string `json:"reason,omitempty"`
	Reference string `json:"reference,omitempty"`
}

// WebhookPayload is sent by the Bank Simulator back to the Payments service webhook endpoint.
type WebhookPayload struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"` // "successful" or "failed"
	Reason    string `json:"reason,omitempty"`
	Reference string `json:"reference"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/authorize", authorizeHandler)

	addr := ":" + port
	log.Printf("bank simulator running on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("bank simulator server error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[Bank Simulator] Received authorization request for payment: %s, amount: %d kobo", req.PaymentID, req.Amount)

	// Simulate bank check rule:
	// If amount ends in 99 kobo, decline due to insufficient_funds
	approved := true
	reason := ""
	reference := fmt.Sprintf("bank_ref_%d", time.Now().UnixNano())

	if req.Amount%100 == 99 {
		approved = false
		reason = "insufficient_funds"
	}

	resp := AuthorizeResponse{
		Approved:  approved,
		Reason:    reason,
		Reference: reference,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

	// Asynchronously dispatch webhook back to Payments Service with retries
	go sendWebhookWithRetries(req.PaymentID, approved, reason, reference)
}

// sendWebhookWithRetries simulates asynchronous bank settlement webhook notifications
func sendWebhookWithRetries(paymentID string, approved bool, reason string, reference string) {
	// Small delay to simulate async network transmission
	time.Sleep(500 * time.Millisecond)

	webhookURL := os.Getenv("PAYMENTS_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "http://localhost:8080/webhooks/bank"
	}

	status := "successful"
	if !approved {
		status = "failed"
	}

	payload := WebhookPayload{
		PaymentID: paymentID,
		Status:    status,
		Reason:    reason,
		Reference: reference,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Bank Simulator Webhook Error] Failed marshaling payload: %v", err)
		return
	}

	maxRetries := 3
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[Bank Simulator Webhook] Sending callback to %s (Attempt %d/%d)...", webhookURL, attempt, maxRetries)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(body))
		if err != nil {
			cancel()
			log.Printf("[Bank Simulator Webhook Error] Failed creating HTTP request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		cancel()

		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("[Bank Simulator Webhook SUCCESS] Webhook delivered to Payments service for payment: %s", paymentID)
			resp.Body.Close()
			return
		}

		if resp != nil {
			resp.Body.Close()
		}

		log.Printf("[Bank Simulator Webhook WARNING] Attempt %d failed. Retrying in %v...", attempt, backoff)
		time.Sleep(backoff)
		backoff *= 2 // Exponential backoff: 500ms, 1s, 2s
	}

	log.Printf("[Bank Simulator Webhook FAILED] Giving up after %d attempts for payment: %s", maxRetries, paymentID)
}
