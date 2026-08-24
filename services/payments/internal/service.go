package payments

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	repo         *Repository
	gateway      BankGateway
	ledgerClient LedgerClient
}

// NewService creates a new Service instance.
func NewService(repo *Repository, gateway BankGateway, ledgerClient LedgerClient) *Service {
	return &Service{
		repo:         repo,
		gateway:      gateway,
		ledgerClient: ledgerClient,
	}
}

// CreatePayment handles the creation of a new payment, enforcing idempotency and state transitions.
func (s *Service) CreatePayment(ctx context.Context, input CreatePaymentInput) (*Payment, error) {

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Idempotency Check: see if a payment with this idempotency key already exists.
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
		Status:         StatusPending,
		CustomerID:     input.CustomerID,
		MerchantID:     input.MerchantID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Save initial "pending" record to database
	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("service: failed to create payment: %w", err)
	}

	// Process payment through Bank Gateway state machine
	processedPayment, err := s.ProcessPayment(ctx, payment.ID)

	if err != nil {
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

	// Mark status as processing
	if err := s.repo.UpdateStatus(ctx, payment.ID, StatusProcessing, ""); err != nil {
		return nil, fmt.Errorf("service: failed updating to processing: %w", err)
	}

	// Authorize payment via Bank Gateway
	approved, failureReason, err := s.gateway.Authorize(ctx, payment)

	// Update status based on bank response
	if approved {
		if err := s.repo.UpdateStatus(ctx, payment.ID, StatusSuccessful, ""); err != nil {
			return nil, fmt.Errorf("service: failed marking payment successful: %w", err)
		}

		// Record entry in ledger
		ledgerInput := LedgerRecordInput{
			ReferenceID: payment.IdempotencyKey,
			CustomerID:  payment.CustomerID,
			MerchantID:  payment.MerchantID,
			Amount:      payment.Amount,
		}

		log.Printf("[PAYMENTS SERVICE] Payment %s successful! Sending accounting entry to Ledger service (:8082)...", payment.ID)

		if err := s.ledgerClient.RecordEntry(ctx, ledgerInput); err != nil {
			log.Printf("[PAYMENTS SERVICE] Failed recording ledger entry: %v", err)
			return nil, fmt.Errorf("service: failed to record ledger entry: %w", err)
		}

		log.Printf("[PAYMENTS SERVICE] Ledger service confirmed accounting entry for payment %s!", payment.ID)
	} else {
		if failureReason == "" && err != nil {
			failureReason = err.Error()
		}
		if errors.Is(err, ErrBankUnreachable) {
			// Bank is offline — revert to pending, worker will retry later
			log.Printf("[PAYMENTS SERVICE] Bank unreachable for payment %s, reverting to pending", payment.ID)
			s.repo.UpdateStatus(ctx, payment.ID, StatusPending, "")
		} else {
			// Bank gave a real decline — mark as failed (terminal)
			if err := s.repo.UpdateStatus(ctx, payment.ID, StatusFailed, failureReason); err != nil {
				return nil, fmt.Errorf("service: failed marking payment failed: %w", err)
			}
		}

	}

	// Fetch updated payment record from repository
	return s.repo.GetByID(ctx, payment.ID)
}

// GetPayment fetches a payment by ID.
func (s *Service) GetPayment(ctx context.Context, id string) (*Payment, error) {
	return s.repo.GetByID(ctx, id)
}
