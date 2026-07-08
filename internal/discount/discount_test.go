package discount_test

import (
	"testing"
	"time"

	"github.com/esrid/maison/internal/discount"
)

func TestComputeDiscountCents(t *testing.T) {
	tests := []struct {
		name          string
		code          discount.Code
		subtotalCents int64
		want          int64
	}{
		{"20 percent of 5000", discount.Code{Type: discount.TypePercent, Value: 20}, 5000, 1000},
		{"100 percent of 5000", discount.Code{Type: discount.TypePercent, Value: 100}, 5000, 5000},
		{"fixed 500 of subtotal 3000", discount.Code{Type: discount.TypeFixed, Value: 500}, 3000, 500},
		{"fixed clamps to subtotal, never negative total", discount.Code{Type: discount.TypeFixed, Value: 5000}, 300, 300},
		{"percent rounds down", discount.Code{Type: discount.TypePercent, Value: 33}, 100, 33},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.code.ComputeDiscountCents(tt.subtotalCents)
			if got != tt.want {
				t.Errorf("ComputeDiscountCents(%d) = %d, want %d", tt.subtotalCents, got, tt.want)
			}
		})
	}
}

func TestCode_IsValid(t *testing.T) {
	now := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		code discount.Code
		want bool
	}{
		{"active, no expiry", discount.Code{Active: true}, true},
		{"active, expires in future", discount.Code{Active: true, ExpiresAt: &future}, true},
		{"active, expired", discount.Code{Active: true, ExpiresAt: &past}, false},
		{"inactive, no expiry", discount.Code{Active: false}, false},
		{"inactive, expires in future", discount.Code{Active: false, ExpiresAt: &future}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.IsValid(now); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
