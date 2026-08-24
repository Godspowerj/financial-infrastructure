package transfers

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Handler handles HTTP requests for transfers.
type Handler struct {
	service *Service
}

// NewHandler constructs a new transfers Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateTransfer handles POST /transfers.
func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input CreateTransferInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	transfer, err := h.service.CreateTransfer(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount),
			errors.Is(err, ErrMissingIdempotencyKey),
			errors.Is(err, ErrMissingSenderID),
			errors.Is(err, ErrMissingReceiverID),
			errors.Is(err, ErrSameSenderReceiver):
			writeJSONError(w, http.StatusBadRequest, err.Error())

		case errors.Is(err, ErrInsufficientFunds):
			writeJSONError(w, http.StatusUnprocessableEntity, err.Error())

		default:
			writeJSONError(w, http.StatusInternalServerError, "failed to process transfer: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, transfer)
}

// GetTransfer handles GET /transfers/{id}.
func (h *Handler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "transfer id is required in path")
		return
	}

	transfer, err := h.service.GetTransfer(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "transfer not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to get transfer: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, transfer)
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

