package discord

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// backoffDelay
// ---------------------------------------------------------------------------

func TestBackoffDelay_FirstAttempt(t *testing.T) {
	d := backoffDelay(0)
	// With jitter of ±20 %, the first attempt should be within [0.8s, 1.2s].
	lo := time.Duration(float64(backoffBase) * (1 - backoffJitter))
	hi := time.Duration(float64(backoffBase) * (1 + backoffJitter))
	if d < lo || d > hi {
		t.Errorf("backoffDelay(0) = %v, want in [%v, %v]", d, lo, hi)
	}
}

func TestBackoffDelay_Grows(t *testing.T) {
	// Each successive attempt (without jitter noise) should be larger than the
	// previous. We test medians by running many samples and checking the mean.
	for attempt := 1; attempt < 10; attempt++ {
		sum := time.Duration(0)
		const samples = 200
		for i := 0; i < samples; i++ {
			sum += backoffDelay(attempt)
		}
		mean := sum / samples
		// Mean should be approximately backoffBase * 2^attempt (capped at backoffMax).
		d := backoffBase
		for i := 0; i < attempt; i++ {
			d = time.Duration(float64(d) * backoffFactor)
			if d > backoffMax {
				d = backoffMax
				break
			}
		}
		// Allow 25 % tolerance on the mean.
		lo := time.Duration(float64(d) * 0.75)
		hi := time.Duration(float64(d) * 1.25)
		if mean < lo || mean > hi {
			t.Errorf("attempt %d: mean backoffDelay = %v, expected ~%v (range [%v, %v])",
				attempt, mean, d, lo, hi)
		}
	}
}

func TestBackoffDelay_CapsAtMax(t *testing.T) {
	// After enough attempts the delay must not exceed backoffMax * (1 + jitter).
	ceiling := time.Duration(float64(backoffMax) * (1 + backoffJitter))
	for i := 0; i < 500; i++ {
		d := backoffDelay(100)
		if d > ceiling {
			t.Errorf("backoffDelay(100) = %v exceeds ceiling %v", d, ceiling)
		}
	}
}
