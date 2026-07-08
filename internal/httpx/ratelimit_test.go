package httpx_test

import (
	"testing"
	"time"

	"github.com/esrid/maison/internal/httpx"
)

func TestRateLimiter(t *testing.T) {
	rl := httpx.NewRateLimiter(3, 50*time.Millisecond)

	for i := 0; i < 3; i++ {
		if !rl.Allow("1.2.3.4") {
			t.Fatalf("attempt %d: want allowed", i+1)
		}
	}
	if rl.Allow("1.2.3.4") {
		t.Fatal("4th attempt within window: want blocked")
	}

	if !rl.Allow("5.6.7.8") {
		t.Fatal("different key should have its own budget")
	}

	time.Sleep(60 * time.Millisecond)
	if !rl.Allow("1.2.3.4") {
		t.Fatal("attempt after window elapsed: want allowed")
	}
}
