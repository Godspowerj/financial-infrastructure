package transfers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Common business logic errors.
var (
	ErrInvalidAmount         = errors.New("amount must be greater than 0")
	ErrMissingIdempotencyKey = errors.New("idempotency_key is required")
	ErrMissingSenderID       = errors.New("sender_id is required")
	ErrMissingReceiverID     = errors.New("receiver_id is required")
	ErrSameSenderReceiver    = errors.New("sender_id and receiver_id cannot be the same")
	ErrInsufficientFunds     = errors.New("sender has insufficient balance")
)

// Service encapsulates business logic and saga coordination for transfers.
type Service struct {
	repo         *Repository
	ledgerClient LedgerClient
}

// NewService constructs a new transfers Service.
func NewService(repo *Repository, ledgerClient LedgerClient) *Service {
	return &Service{
		repo:         repo,
		ledgerClient: ledgerClient,
	}
}

// CreateTransfer coordinates the saga: validating input, checking balance, and recording double-entry in Ledger.
func (s *Service) CreateTransfer(ctx context.Context, input CreateTransferInput) (*Transfer, error) {
	//Implement input validation
	if input.SenderID == input.ReceiverID {
		return nil, ErrSameSenderReceiver
	}
	if input.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if input.IdempotencyKey == "" {
		return nil, ErrMissingIdempotencyKey
	}
	if input.SenderID == "" {
		return nil, ErrMissingSenderID
	}
	if input.ReceiverID == "" {
		return nil, ErrMissingReceiverID
	}

	//Implement idempotency check
	existing, err := s.repo.GetByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("service: failed checking idempotency: %w", err)
	}
	// Implement balance check
	senderBalance, err := s.ledgerClient.GetBalance(ctx, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("service: failed checking sender balance: %w", err)
	}
	if senderBalance < input.Amount {
		return nil, ErrInsufficientFunds
	}
	//Implement transfer record creation
	now := time.Now().UTC()
	t := &Transfer{
		ID:             uuid.New().String(),
		IdempotencyKey: input.IdempotencyKey,
		SenderID:       input.SenderID,
		ReceiverID:     input.ReceiverID, 
		Amount:         input.Amount,
		Currency:       "NGN",
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("service: failed creating transfer record: %w", err)
	}
	//Implement ledger double-entry
	if err := s.ledgerClient.PostTransferEntry(ctx, t.ID, t.SenderID, t.ReceiverID, t.Amount); err != nil {
		s.repo.UpdateStatus(ctx, t.ID, StatusFailed, "ledger_unavailable")
		return nil, fmt.Errorf("service: failed posting ledger entry: %w", err)
	}
	//Update transfer status
	t.Status = StatusProcessed
	if err := s.repo.UpdateStatus(ctx, t.ID, t.Status, ""); err != nil {
		return nil, fmt.Errorf("service: failed updating transfer status to processed	: %w", err)
	}
	return t, nil
}

// GetTransfer fetches a transfer by ID.
func (s *Service) GetTransfer(ctx context.Context, id string) (*Transfer, error) {
	if id == "" {
		return nil, errors.New("transfer id is required")
	}
	return s.repo.GetByID(ctx, id)
}
