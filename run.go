package replicate

import (
	"context"
	"errors"
)

func (r *Client) Run(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (PredictionOutput, error) {
	id, err := ParseIdentifier(identifier)
	if err != nil {
		return nil, err
	}

	if id.Version == nil {
		return nil, errors.New("version must be specified")
	}

	prediction, err := r.CreatePrediction(ctx, *id.Version, input, webhook, false)
	if err != nil {
		return nil, err
	}

	err = r.Wait(ctx, prediction)

	return prediction.Output, err
}
