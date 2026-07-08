// Package payment defines the boundary between checkout and whatever
// actually moves money. MockProvider always succeeds — it exists so
// checkout has something to depend on today; swap in a real Stripe HTTP
// client behind Provider later without touching checkout at all.
package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type Receipt struct {
	Status    string
	Reference string
}

type Provider interface {
	Charge(ctx context.Context, amountCents int64, description string) (Receipt, error)
}

type MockProvider struct{}

func (MockProvider) Charge(_ context.Context, _ int64, _ string) (Receipt, error) {
	ref, err := randomRef()
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{Status: "paid", Reference: ref}, nil
}

func randomRef() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mock_" + hex.EncodeToString(b), nil
}
