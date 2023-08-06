package replicate

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultInterval = 1 * time.Second
)

type waitOptions struct {
	interval    time.Duration
	maxAttempts *int
}

// WaitOption is a function that modifies an options struct.
type WaitOption func(*waitOptions) error

// WithInterval sets the interval between attempts.
func WithInterval(interval time.Duration) WaitOption {
	return func(o *waitOptions) error {
		o.interval = interval
		return nil
	}
}

// WithMaxAttempts sets the maximum number of attempts.
func WithMaxAttempts(maxAttempts int) WaitOption {
	return func(o *waitOptions) error {
		o.maxAttempts = &maxAttempts
		return nil
	}
}

// Wait for a prediction to finish.
//
// This function blocks until the prediction has finished, or the context is cancelled.
// If the prediction has already finished, the function returns immediately.
// If the prediction has not finished after maxAttempts, an error is returned.
// If interval is less than or equal to zero, an error is returned.
// If maxAttempts is less than zero, an error is returned.
// If maxAttempts is equal to zero, there is no limit to the number of attempts.
func (r *Client) Wait(ctx context.Context, prediction *Prediction, opts ...WaitOption) error {
	options := &waitOptions{
		interval: defaultInterval,
	}

	for _, option := range opts {
		err := option(options)
		if err != nil {
			return err
		}
	}

	ticker := time.NewTicker(options.interval)
	defer ticker.Stop()

	id := prediction.ID
	attempts := 0
	for {
		select {
		case <-ticker.C:
			updatedPrediction, err := r.GetPrediction(ctx, id)
			if err != nil {
				return err
			}

			*prediction = *updatedPrediction

			if prediction.Status.Terminated() {
				return nil
			}

			attempts += 1
			if options.maxAttempts != nil && attempts >= *options.maxAttempts {
				return fmt.Errorf("prediction %s did not finish after %d attempts", id, *options.maxAttempts)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
