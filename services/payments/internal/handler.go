package payments

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// Handler handles HTTP requests for payments.
type Handler struct {
	service *Service
}

// NewHandler creates a new Handler instance.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreatePayment handles POST /payments
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input CreatePaymentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	payment, err := h.service.CreatePayment(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount),
			errors.Is(err, ErrMissingIdempotencyKey),
			errors.Is(err, ErrMissingCustomerID),
			errors.Is(err, ErrMissingMerchantID):
			writeJSONError(w, http.StatusBadRequest, err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, "failed to process payment: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, payment)
}

// GetPayment handles GET /payments/{id}
func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path: /payments/{id}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[1] == "" {
		writeJSONError(w, http.StatusBadRequest, "payment id is required in path")
		return
	}
	id := pathParts[1]

	payment, err := h.service.GetPayment(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "payment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch payment: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, payment)
}

// ProcessPayment handles POST /payments/{id}/process
func (h *Handler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[1] == "" {
		writeJSONError(w, http.StatusBadRequest, "payment id is required in path")
		return
	}
	id := pathParts[1]

	payment, err := h.service.ProcessPayment(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "payment not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to process payment: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, payment)
}

// BankWebhookPayload is the incoming payload received on POST /webhooks/bank
type BankWebhookPayload struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	Reference string `json:"reference"`
}

// HandleBankWebhook handles POST /webhooks/bank callbacks from the Bank Simulator.
func (h *Handler) HandleBankWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload BankWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid webhook JSON payload: "+err.Error())
		return
	}

	// Update payment status in Postgres based on async bank settlement webhook
	err := h.service.repo.UpdateStatus(r.Context(), payload.PaymentID, payload.Status, payload.Reason)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "payment for webhook not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed updating status from webhook: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "acknowledged",
		"payment_id": payload.PaymentID,
	})
}

// Helper functions for clean JSON HTTP responses
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
