package transfers

import "time"

// --- Transfer Status Constants ---
// These define the allowed states in the transfer lifecycle.
// Using string constants (not a DB ENUM) so the state machine
// can evolve freely in Go code before being locked down.

const (
	StatusPending    = "pending"    // Transfer created, not yet started
	StatusProcessing = "processing" // Saga steps in progress
	StatusCompleted  = "completed"  // All saga steps succeeded
	StatusFailed     = "failed"     // A step failed, compensation may have run
	StatusReversed   = "reversed"   // Compensation completed after failure
)

// Transfer represents a money movement request between two parties.
// The Transfers service coordinates the saga across Ledger and
// potentially other services — it doesn't move money directly.
type Transfer struct {
	ID             string    `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	SenderID       string    `json:"sender_id"`
	ReceiverID     string    `json:"receiver_id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	FailureReason  string    `json:"failure_reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IsTerminal returns true if the transfer has reached a final state
// and cannot transition further. Same guard pattern used in Payments.
func (t *Transfer) IsTerminal() bool {
	return t.Status == StatusCompleted ||
		t.Status == StatusFailed ||
		t.Status == StatusReversed
}

// CreateTransferInput is the data needed from an API caller
// to initiate a new transfer.
type CreateTransferInput struct {
	IdempotencyKey string `json:"idempotency_key"`
	SenderID       string `json:"sender_id"`
	ReceiverID     string `json:"receiver_id"`
	Amount         int64  `json:"amount"`
}
