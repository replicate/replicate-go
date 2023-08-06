package replicate

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
)

const defaultPollingInterval = 1 * time.Second

func (r *Client) Run(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (PredictionOutput, error) {
	namePattern := `[a-zA-Z0-9]+(?:(?:[._]|__|[-]*)[a-zA-Z0-9]+)*`
	pattern := fmt.Sprintf(`^(?P<owner>%s)/(?P<name>%s):(?P<version>[0-9a-fA-F]+)$`, namePattern, namePattern)

	regex := regexp.MustCompile(pattern)
	match := regex.FindStringSubmatch(identifier)

	if len(match) == 0 {
		return nil, errors.New("invalid version. it must be in the format \"owner/name:version\"")
	}

	version := ""
	for i, name := range regex.SubexpNames() {
		if name == "version" {
			version = match[i]
		}
	}

	prediction, err := r.CreatePrediction(ctx, version, input, webhook, false)
	if err != nil {
		return nil, err
	}

	r.Wait(ctx, prediction)

	return prediction.Output, err
}
