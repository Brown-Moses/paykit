package auth

import (
	"errors"
	"testing"
)

// MockSeenChecker implements the SeenChecker interface for testing
type MockSeenChecker struct {
	existsMap map[string]bool
	err       error
}

func (m *MockSeenChecker) Exists(providerTxID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.existsMap[providerTxID], nil
}

func TestVerifier_Verify(t *testing.T) {
	secret := "my_momo_secret_key"
	body := []byte(`{"transactionId":"TX-100","externalId":"order-abc","amount":"1000"}`)
	validSignature := ComputeSignature(secret, body)

	tests := []struct {
		name         string
		secret       string
		body         []byte
		signature    string
		providerTxID string
		seenMap      map[string]bool
		seenErr      error
		expectedErr  error
	}{
		{
			name:         "Success",
			secret:       secret,
			body:         body,
			signature:    validSignature,
			providerTxID: "TX-100",
			seenMap:      map[string]bool{},
			expectedErr:  nil,
		},
		{
			name:         "Missing Signature",
			secret:       secret,
			body:         body,
			signature:    "",
			providerTxID: "TX-100",
			seenMap:      map[string]bool{},
			expectedErr:  ErrMissingSignature,
		},
		{
			name:         "Invalid Signature (Forged)",
			secret:       secret,
			body:         body,
			signature:    "invalid_signature_here",
			providerTxID: "TX-100",
			seenMap:      map[string]bool{},
			expectedErr:  ErrInvalidSignature,
		},
		{
			name:         "Replay Attack (Already Processed)",
			secret:       secret,
			body:         body,
			signature:    validSignature,
			providerTxID: "TX-100",
			seenMap:      map[string]bool{"TX-100": true},
			expectedErr:  ErrReplay,
		},
		{
			name:         "Empty Provider Tx ID",
			secret:       secret,
			body:         body,
			signature:    validSignature,
			providerTxID: "",
			seenMap:      map[string]bool{},
			expectedErr:  ErrMissingProviderTxID,
		},
		{
			name:         "Seen Checker Failure",
			secret:       secret,
			body:         body,
			signature:    validSignature,
			providerTxID: "TX-100",
			seenMap:      map[string]bool{},
			seenErr:      errors.New("db disconnect"),
			expectedErr:  ErrReplayFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := &MockSeenChecker{
				existsMap: tt.seenMap,
				err:       tt.seenErr,
			}
			v := NewVerifier(tt.secret, seen)
			err := v.Verify(tt.body, tt.signature, tt.providerTxID)

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Verifier.Verify() error = %v, expectedErr = %v", err, tt.expectedErr)
			}
		})
	}
}
