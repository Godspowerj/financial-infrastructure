package transfers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("transfer not found")

// Repository handles database operations for transfers in PostgreSQL.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new transfers Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new transfer record in Postgres.
func (r *Repository) Create(ctx context.Context, t *Transfer) error {
	const query = `
		INSERT INTO transfers (id, idempotency_key, sender_id, receiver_id, amount, currency, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query, 
		t.ID,
		t.IdempotencyKey,
		t.SenderID,
		t.ReceiverID,
		t.Amount,
		t.Currency,
		t.Status,
		t.CreatedAt,
		t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: failed to insert transfer: %w", err)
	}

	return nil
}

// GetByID fetches a transfer by its UUID primary key.
func (r *Repository) GetByID(ctx context.Context, id string) (*Transfer, error) {
	const query = `
		SELECT id, idempotency_key, sender_id, receiver_id, amount, currency,
		       status, COALESCE(failure_reason, ''), created_at, updated_at
		FROM transfers
		WHERE id = $1
	`
	var t Transfer
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.IdempotencyKey,
		&t.SenderID,
		&t.ReceiverID,
		&t.Amount,
		&t.Currency,
		&t.Status,
		&t.FailureReason,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository: failed to get transfer by ID: %w", err)
	}

	return &t, nil
}

// GetByIdempotencyKey fetches a transfer by its unique idempotency key.
func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (*Transfer, error) {
	const query = `
		SELECT id, idempotency_key, sender_id, receiver_id, amount, currency,
		       status, COALESCE(failure_reason, ''), created_at, updated_at
		FROM transfers
		WHERE idempotency_key = $1
	`
	var t Transfer
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&t.ID,
		&t.IdempotencyKey,
		&t.SenderID,
		&t.ReceiverID,
		&t.Amount,
		&t.Currency,
		&t.Status,
		&t.FailureReason,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository: failed to get transfer by idempotency key: %w", err)
	}

	return &t, nil
}

// UpdateStatus updates the lifecycle status and optional failure reason of a transfer.
func (r *Repository) UpdateStatus(ctx context.Context, id string, status string, failureReason string) error {
	const query = `
		UPDATE transfers
		SET status = $1,
		    failure_reason = NULLIF($2, ''),
		    updated_at = NOW()
		WHERE id = $3
	`
	res, err := r.db.ExecContext(ctx, query, status, failureReason, id)
	if err != nil {
		return fmt.Errorf("repository: failed to update transfer status: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

