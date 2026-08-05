package payments

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
	ErrMissingCustomerID     = errors.New("customer_id is required")
	ErrMissingMerchantID     = errors.New("merchant_id is required")
)

// Service encapsulates business logic for payments.
type Service struct {
	repo    *Repository
	gateway BankGateway
}

// NewService creates a new Service instance.
func NewService(repo *Repository, gateway BankGateway) *Service {
	return &Service{
		repo:    repo,
		gateway: gateway,
	}
}

// CreatePayment handles the creation of a new payment, enforcing idempotency and state transitions.
func (s *Service) CreatePayment(ctx context.Context, input CreatePaymentInput) (*Payment, error) {
	// 1. Basic business validation
	if input.IdempotencyKey == "" {
		return nil, ErrMissingIdempotencyKey
	}
	if input.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if input.CustomerID == "" {
		return nil, ErrMissingCustomerID
	}
	if input.MerchantID == "" {
		return nil, ErrMissingMerchantID
	}

	// 2. Idempotency Check: see if a payment with this idempotency key already exists.
	existing, err := s.repo.GetByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil {
		// Payment already exists — return the previously created payment safely (idempotent response)
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("service: failed checking idempotency: %w", err)
	}

	// Default currency to NGN if empty
	currency := input.Currency
	if currency == "" {
		currency = "NGN"
	}

	now := time.Now().UTC()
	payment := &Payment{
		ID:             uuid.New().String(),
		IdempotencyKey: input.IdempotencyKey,
		Amount:         input.Amount,
		Currency:       currency,
		Status:         StatusPending, // 1. Record intent as pending
		CustomerID:     input.CustomerID,
		MerchantID:     input.MerchantID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// 3. Save initial "pending" record to database
	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("service: failed to create payment: %w", err)
	}

	// 4. Process payment through Bank Gateway state machine
	processedPayment, err := s.ProcessPayment(ctx, payment.ID)
	if err != nil {
		// Even if processing returns an error, return the latest payment state from DB
		return payment, nil
	}

	return processedPayment, nil
}

// ProcessPayment transitions a payment from pending -> processing -> successful/failed by communicating with BankGateway.
func (s *Service) ProcessPayment(ctx context.Context, id string) (*Payment, error) {
	payment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If payment is already in a terminal state (successful or failed), do not re-process
	if payment.IsTerminal() {
		return payment, nil
	}

	// Step A: Mark status as processing
	if err := s.repo.UpdateStatus(ctx, payment.ID, StatusProcessing, ""); err != nil {
		return nil, fmt.Errorf("service: failed updating to processing: %w", err)
	}

	// Step B: Authorize payment via Bank Gateway
	approved, failureReason, err := s.gateway.Authorize(ctx, payment)

	// Step C: Update status based on bank response
	if approved {
		if err := s.repo.UpdateStatus(ctx, payment.ID, StatusSuccessful, ""); err != nil {
			return nil, fmt.Errorf("service: failed marking payment successful: %w", err)
		}
	} else {
		if failureReason == "" && err != nil {
			failureReason = err.Error()
		}
		if err := s.repo.UpdateStatus(ctx, payment.ID, StatusFailed, failureReason); err != nil {
			return nil, fmt.Errorf("service: failed marking payment failed: %w", err)
		}
	}

	// Fetch updated payment record from repository
	return s.repo.GetByID(ctx, payment.ID)
}

// GetPayment fetches a payment by ID.
func (s *Service) GetPayment(ctx context.Context, id string) (*Payment, error) {
	return s.repo.GetByID(ctx, id)
}
