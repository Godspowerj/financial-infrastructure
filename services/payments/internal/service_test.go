package payments

import (
	"context"
	"testing"
)

// TODO: Challenge 1 - Test successful payment processing state transition
func TestProcessPayment_Successful(t *testing.T) {
	// 1. Arrange: Create mock dependencies and a test payment struct
	mockGateway := &MockBankGateway{
		ShouldApprove: true,
	}

	ctx := context.Background()

	// 2. Act: Call service.ProcessPayment(...)
	payment := &Payment{ID: "pay_123_test", Amount: 4000}
	approved, _, err := mockGateway.Authorize(ctx, payment)

	// 3. Assert: Verify payment status is StatusSuccessful ("successful")

	if !approved {
		t.Errorf("expected payment to be approved, got %v", approved)
	}

	if err != nil {
		t.Errorf("expected no error for payment, got %v", err)
	}

}

// TODO: Challenge 2 - Test failed payment processing when bank declines
func TestProcessPayment_Declined(t *testing.T) {
	// 1. Arrange: Create mock bank returning (false, "insufficient_funds", nil)
	mockGateway := &MockBankGateway{
		ShouldApprove: false,
		Reason:        "insufficient_funds",
		Err:           nil,
	}

	ctx := context.Background()

	// 2. Act: Call service.ProcessPayment(...)
	payment := &Payment{ID: "pay_123_test", Amount: 4000}
	approved, reason, err := mockGateway.Authorize(ctx, payment)

	// 3. Assert: Verify payment status is StatusFailed ("failed") and failure_reason is "insufficient_funds"

	if approved {
		t.Errorf("expected payment to be declined, got %v", approved)
	}

	if reason != "insufficient_funds" {
		t.Errorf("expected failure reason 'insufficient_funds', got %s", reason)
	}

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
