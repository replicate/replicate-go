package replicate

import (
	"math"
	"math/rand"
	"time"
)

// Backoff is an interface for backoff strategies.
type Backoff interface {
	NextDelay(retries int) time.Duration
}

// ConstantBackoff is a backoff strategy that returns a constant delay with some jitter.
type ConstantBackoff struct {
	Base   time.Duration
	Jitter time.Duration
}

// NextDelay returns the next delay.
func (b *ConstantBackoff) NextDelay(_ int) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(b.Jitter)) //#nosec G404
	return b.Base + jitter
}

// ExponentialBackoff is a backoff strategy that returns an exponentially increasing delay with some jitter.
type ExponentialBackoff struct {
	Base       time.Duration
	Multiplier float64
	Jitter     time.Duration
}

// NextDelay returns the next delay.
func (b *ExponentialBackoff) NextDelay(retries int) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(b.Jitter)) //#nosec G404
	return time.Duration(float64(b.Base)*math.Pow(b.Multiplier, float64(retries))) + jitter
}
