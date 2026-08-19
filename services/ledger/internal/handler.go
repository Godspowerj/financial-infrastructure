package ledger

import (
	"encoding/json"
	"log"
	"net/http"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) CreateEntryWithLines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input RecordPaymentInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	log.Printf("📥 [LEDGER SERVICE] Received accounting request! ref=%s, customer=%s, merchant=%s, amount=%d kobo", 
		input.ReferenceID, input.CustomerID, input.MerchantID, input.Amount)

	if input.Amount <= 0 {
		http.Error(w, "Invalid Amount: must be greater than 0", http.StatusBadRequest)
		return
	}
	if input.CustomerID == "" {
		http.Error(w, "Invalid Customer ID", http.StatusBadRequest)
		return
	}
	if input.ReferenceID == "" {
		http.Error(w, "Invalid Reference ID", http.StatusBadRequest)
		return
	}
	if input.MerchantID == "" {
		http.Error(w, "Invalid Merchant ID", http.StatusBadRequest)
		return
	}

	entry, err := h.Service.RecordPayment(r.Context(), input)
	if err != nil {
		log.Printf("❌ [LEDGER SERVICE] Failed to record entry: %v", err)
		http.Error(w, "Failed to create entry with lines: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ [LEDGER SERVICE] Successfully recorded double-entry accounting lines for ref=%s!", entry.ReferenceID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func (h *Handler) GetLedgerEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	entries, err := h.Service.GetLedgerEntries(r.Context())
	if err != nil {
		log.Printf("❌ [LEDGER SERVICE] Failed to get ledger entries: %v", err)
		http.Error(w, "Failed to get ledger entries: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entries)
}
