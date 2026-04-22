package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateAPIKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate api key: %w", err)
	}

	return "pk_live_" + hex.EncodeToString(b), nil
}
