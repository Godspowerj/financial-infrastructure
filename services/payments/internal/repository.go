package payments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Common errors returned by the repository layer.
var ErrNotFound = errors.New("payment not found")

// Repository manages database operations for payments.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Repository instance.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new payment record into Postgres.
func (r *Repository) Create(ctx context.Context, p *Payment) error {
	query := `
		INSERT INTO payments (
			id, idempotency_key, amount, currency, status, failure_reason, customer_id, merchant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		p.ID,
		p.IdempotencyKey,
		p.Amount,
		p.Currency,
		p.Status,
		p.FailureReason,
		p.CustomerID,
		p.MerchantID,
		p.CreatedAt,
		p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: failed to insert payment: %w", err)
	}

	return nil
}

// UpdateStatus updates the status, failure reason, and updated_at timestamp of a payment record.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status string, failureReason string) error {
	query := `
		UPDATE payments
		SET status = $1, failure_reason = $2, updated_at = NOW()
		WHERE id = $3
	`

	res, err := r.db.ExecContext(ctx, query, status, failureReason, id)
	if err != nil {
		return fmt.Errorf("repository: failed to update payment status: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: failed getting rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetByID fetches a payment by its unique UUID string.
func (r *Repository) GetByID(ctx context.Context, id string) (*Payment, error) {
	query := `
		SELECT id, idempotency_key, amount, currency, status, COALESCE(failure_reason, ''), customer_id, merchant_id, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	var payment Payment
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&payment.ID,
		&payment.IdempotencyKey,
		&payment.Amount,
		&payment.Currency,
		&payment.Status,
		&payment.FailureReason,
		&payment.CustomerID,
		&payment.MerchantID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository: failed to query payment by id: %w", err)
	}

	return &payment, nil
}

// GetByIdempotencyKey fetches an existing payment by its idempotency key.
func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (*Payment, error) {
	query := `
		SELECT id, idempotency_key, amount, currency, status, COALESCE(failure_reason, ''), customer_id, merchant_id, created_at, updated_at
		FROM payments
		WHERE idempotency_key = $1
	`

	var p Payment
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&p.ID,
		&p.IdempotencyKey,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&p.FailureReason,
		&p.CustomerID,
		&p.MerchantID,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository: failed to query payment by idempotency key: %w", err)
	}

	return &p, nil
}

// GetRetryablePayments fetches all pending payments in the system within the 30-minute window.
func (r *Repository) GetRetryablePayments(ctx context.Context) ([]Payment, error) {
	const query = `
		SELECT id, idempotency_key, amount, currency, status, COALESCE(failure_reason, ''),
		       customer_id, merchant_id, created_at, updated_at
		FROM payments
		WHERE status = 'pending'
		  AND created_at > NOW() - INTERVAL '30 minutes'
		ORDER BY created_at ASC
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to query retryable payments: %w", err)
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(
			&p.ID,
			&p.IdempotencyKey,
			&p.Amount,
			&p.Currency,
			&p.Status,
			&p.FailureReason,
			&p.CustomerID,
			&p.MerchantID,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: failed to scan payment: %w", err)
		}
		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: error iterating retryable payments: %w", err)
	}

	return payments, nil
}
