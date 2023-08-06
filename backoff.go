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

type ConstantBackoff struct {
	Base   time.Duration
	Jitter time.Duration
}

func (b *ConstantBackoff) NextDelay(retries int) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(b.Jitter))
	return b.Base + jitter
}

type ExponentialBackoff struct {
	Base       time.Duration
	Multiplier float64
	Jitter     time.Duration
}

func (b *ExponentialBackoff) NextDelay(retries int) time.Duration {
	jitter := time.Duration(rand.Float64() * float64(b.Jitter))
	return time.Duration(float64(b.Base)*math.Pow(b.Multiplier, float64(retries))) + jitter
}
