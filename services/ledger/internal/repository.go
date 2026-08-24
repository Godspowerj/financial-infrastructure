package ledger


import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// create account registers a finanacial bucket in the account table. e.g customers account

func (r *Repository) CreateAccount(ctx context.Context, acc Account	) error {
	query := `
	INSERT INTO accounts (id, type, currency, created_at)
	VALUES ($1, $2, $3, now())
	`

	_, err := r.db.ExecContext(ctx, query, acc.ID, acc.Type, acc.Currency)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

func (r *Repository) GetAccount(ctx context.Context, id string) (Account, error) {
	query := `
	SELECT id, type, currency, created_at
	FROM accounts
	WHERE id = $1
	`
	var acc Account
	err := r.db.QueryRowContext(ctx, query, id).Scan(&acc.ID, &acc.Type, &acc.Currency, &acc.CreatedAt)
	if err != nil {
		return Account{}, fmt.Errorf("failed to get account: %w", err)
	}
	return acc, nil
}

func (r *Repository) UpdateAccount(ctx context.Context, acc Account) error {
	query := `
	UPDATE accounts
	SET type = $1, currency = $2, created_at = $3
	WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, acc.Type, acc.Currency, acc.CreatedAt, acc.ID)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}
	return nil
} 

func (r *Repository) GetLedgerLines(ctx context.Context, entryID string) ([]LedgerLine, error) {
	query := `
	SELECT id, entry_id, account_id, amount, direction, created_at
	FROM ledger_lines
	WHERE entry_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger lines: %w", err)
	}
	defer rows.Close()

	var lines []LedgerLine
	for rows.Next() {
		var line LedgerLine
		if err := rows.Scan(&line.ID, &line.EntryID, &line.AccountID, &line.Amount, &line.Direction, &line.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ledger line: %w", err)
		}
		lines = append(lines, line)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during lines iteration: %w", err)
	}

	return lines, nil
}

// CreateEntryWithLines atomically inserts a ledger entry and all its debit/credit lines in a single SQL transaction
func (r *Repository) CreateEntryWithLines(ctx context.Context, entry LedgerEntry, lines []LedgerLine) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // automatically rolls back if an error occurs

	// Insert Entry Envelope
	entryQuery := `
		INSERT INTO ledger_entries (reference_id)
		VALUES ($1)
		RETURNING id, created_at
	`
	err = tx.QueryRowContext(ctx, entryQuery, entry.ReferenceID).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert ledger entry: %w", err)
	}

	// Insert each Line Slip using the entry.ID
	lineQuery := `
		INSERT INTO ledger_lines (entry_id, account_id, amount, direction)
		VALUES ($1, $2, $3, $4)
	`
	for _, line := range lines {
		_, err := tx.ExecContext(ctx, lineQuery, entry.ID, line.AccountID, line.Amount, line.Direction)
		if err != nil {
			return fmt.Errorf("failed to insert ledger line: %w", err)
		}
	}

	// Commit everything to disk atomically!
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}



func (r *Repository) GetAllLedgerEntries(ctx context.Context) ([]LedgerEntry, error) {
	query := `
	SELECT id, reference_id, created_at
	FROM ledger_entries
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []LedgerEntry
	for rows.Next() {
		var entry LedgerEntry
		if err := rows.Scan(&entry.ID, &entry.ReferenceID, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during entries iteration: %w", err)
	}

	return entries, nil
}

func (r *Repository) GetLedgerEntryByID(ctx context.Context, id string) (*LedgerEntry, error) {
	query := `
	SELECT id, reference_id, created_at
	FROM ledger_entries
	WHERE id = $1
	`
	var entry LedgerEntry
	err := r.db.QueryRowContext(ctx, query, id).Scan(&entry.ID, &entry.ReferenceID, &entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entry: %w", err)
	}
	return &entry, nil
}

// GetAccountBalance calculates the net balance for an account (credits - debits).
func (r *Repository) GetAccountBalance(ctx context.Context, accountID string) (int64, error) {
	const query = `
		SELECT COALESCE(
			SUM(CASE WHEN direction = 'credit' THEN amount ELSE -amount END),
			0
		)
		FROM ledger_lines
		WHERE account_id = $1
	`
	var balance int64
	err := r.db.QueryRowContext(ctx, query, accountID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate account balance: %w", err)
	}

	return balance, nil
}
