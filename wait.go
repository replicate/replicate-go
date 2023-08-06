package replicate

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Wait for a prediction to finish.
//
// This function blocks until the prediction has finished, or the context is cancelled.
// If the prediction has already finished, the prediction is returned immediately.
// If the prediction has not finished after maxAttempts, an error is returned.
// If interval is less than or equal to zero, an error is returned.
// If maxAttempts is 0, there is no limit to the number of attempts.
// If maxAttempts is negative, an error is returned.
func (r *Client) Wait(ctx context.Context, prediction Prediction, interval time.Duration, maxAttempts int) (*Prediction, error) {
	if prediction.Status.Terminated() {
		return &prediction, nil
	}

	if interval <= 0 {
		return nil, errors.New("interval must be greater than zero")
	}

	if maxAttempts < 0 {
		return nil, errors.New("maxAttempts must be greater than or equal to zero")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	id := prediction.ID

	attempts := 0
	for {
		select {
		case <-ticker.C:
			prediction, err := r.GetPrediction(ctx, id)
			if err != nil {
				return nil, err
			}

			if prediction.Status.Terminated() {
				return prediction, nil
			}

			attempts += 1
			if maxAttempts > 0 && attempts > maxAttempts {
				return nil, fmt.Errorf("prediction %s did not finish after %d attempts", id, maxAttempts)
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
