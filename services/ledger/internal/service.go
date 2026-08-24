package ledger

import (
	"context"
	"errors"
)

type Service struct {
	repo *Repository
}

var (
	ErrInvalidAmount      = errors.New("amount must be greater than 0")
	ErrMissingReferenceID = errors.New("reference_id is required")
	ErrMissingCustomerID  = errors.New("customer_id is required")
	ErrMissingMerchantID  = errors.New("merchant_id is required")
)

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) RecordPayment(ctx context.Context, input RecordPaymentInput) (*LedgerEntry, error) {
	if input.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if input.ReferenceID == "" {
		return nil, ErrMissingReferenceID
	}
	if input.CustomerID == "" {
		return nil, ErrMissingCustomerID
	}
	if input.MerchantID == "" {
		return nil, ErrMissingMerchantID
	}

	customerAccID := "customer:" + input.CustomerID
	merchantAccID := "merchant:" + input.MerchantID

	//Ensure both financial accounts exist and abort if not.
	err := s.repo.CreateAccount(ctx, Account{ID: customerAccID, Type: "asset", Currency: "NGN"})
	if err != nil {
		return nil, err
	}
	err = s.repo.CreateAccount(ctx, Account{ID: merchantAccID, Type: "liability", Currency: "NGN"})
	if err != nil {
		return nil, err
	}

	//The Split Payment: Customer assets decrease (credit), merchant liabilities increase (debit)

	entry := LedgerEntry{
		ReferenceID: input.ReferenceID,
	}

	lines := []LedgerLine{
		{
			AccountID: customerAccID,
			Amount:    input.Amount,
			Direction: "debit",
		},
		{
			AccountID: merchantAccID,
			Amount:    input.Amount,
			Direction: "credit",
		},
	}

	if err := s.repo.CreateEntryWithLines(ctx, entry, lines); err != nil {
		return nil, err
	}

	return &entry, nil

}
func (s *Service) GetLedgerEntries(ctx context.Context) ([]LedgerEntry, error) {
	return s.repo.GetAllLedgerEntries(ctx)
}
func (s *Service) GetLedgerEntryByID(ctx context.Context, id string) (*LedgerEntry, error) {
	return s.repo.GetLedgerEntryByID(ctx, id)
}

func (s *Service) GetAccountBalance(ctx context.Context, accountID string) (int64, error) {
	if accountID == "" {
		return 0, errors.New("account_id is required")
	}
	return s.repo.GetAccountBalance(ctx, accountID)
}
