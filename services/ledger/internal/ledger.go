package ledger

import "time"

type Account struct {
	ID  string  `json:"id"`
	Type  string `json:"type"`
	Currency string `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}


type LedgerEntry struct {
	ID string `json:"id"`
	ReferenceID string `json:"reference_id"`
	CreatedAt time.Time `json:"created_at"`
}


type RecordPaymentInput struct {
	ReferenceID string `json:"reference_id"`
	CustomerID  string `json:"customer_id"`
	MerchantID  string `json:"merchant_id"`
	Amount      int64  `json:"amount"`
}

type LedgerLine struct {
	ID string `json:"id"`
	EntryID string `json:"entry_id"`
	AccountID string `json:"account_id"`
	Amount int64 `json:"amount"`
	Direction string `json:"direction"`
	CreatedAt time.Time `json:"created_at"`
}
	

