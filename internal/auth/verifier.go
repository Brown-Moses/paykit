package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrMissingSignature    = errors.New("x-signature header is missing")
	ErrInvalidSignature    = errors.New("signature does not match - possible forged request")
	ErrReplay              = errors.New("transaction ID already processed - possible replay attack")
	ErrMissingProviderTxID = errors.New("providerTxID is empty — cannot check replay")
	ErrReplayFailure       = errors.New("replay check failed")
)

type Verifier struct {
	secret []byte
	seen   SeenChecker
}

func NewVerifier(secret string, seen SeenChecker) *Verifier {
	return &Verifier{
		secret: []byte(secret),
		seen:   seen,
	}

}

type SeenChecker interface {
	Exists(providerTxID string) (bool, error)
}

// verify both checks
func (v *Verifier) Verify(body []byte, signatureHeader string, providerTxID string) error {
	if err := v.verifySignature(body, signatureHeader); err != nil {
		return err
	}
	if err := v.checkReplay(providerTxID); err != nil {
		return err
	}

	return nil
}

func (v *Verifier) verifySignature(body []byte, signatureHeader string) error {

	if signatureHeader == "" {
		return ErrMissingSignature
	}

	mac := hmac.New(sha256.New, v.secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	//hmac.Equal does a constant-time comparison
	//Donot use ' == ' because timing attacks can leak the secret byte by byte
	if !hmac.Equal([]byte(expected), []byte(signatureHeader)) {
		return ErrInvalidSignature
	}

	return nil
}

// check if transaction is stored
func (v *Verifier) checkReplay(providerTxID string) error {
	if providerTxID == "" {
		return ErrMissingProviderTxID
	}

	exists, err := v.seen.Exists(providerTxID)
	if err != nil {
		return ErrReplayFailure
	}
	if exists {
		return ErrReplay
	}

	return nil
}

// test helper
func ComputeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
