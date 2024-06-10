package replicate_test

import (
	"context"
	"fmt"

	"github.com/replicate/replicate-go"
)

func ExampleClient_Run() {
	ctx := context.TODO()

	// You can also provide a token directly with `replicate.NewClient(replicate.WithToken("r8_..."))`
	r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
	if err != nil {
		panic(err)
	}

	model := "bytedance/sdxl-lightning-4step"
	version := "5f24084160c9089501c1b3545d9be3c27883ae2239b6f412990e82d4a6210f8f"

	input := replicate.PredictionInput{
		"prompt": "An astronaut riding a rainbow unicorn",
	}

	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}

	// Run a model and wait for its output
	output, err := r8.Run(ctx, fmt.Sprintf("%s:%s", model, version), input, &webhook)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated %d image(s)\n", len(output.([]any)))
	// Output: Generated 1 image(s)
}

func ExampleClient_CreatePrediction() {
	ctx := context.Background()

	// You can also provide a token directly with `replicate.NewClient(replicate.WithToken("r8_..."))`
	r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
	if err != nil {
		panic(err)
	}

	// https://replicate.com/bytedance/sdxl-lightning-4step
	version := "5f24084160c9089501c1b3545d9be3c27883ae2239b6f412990e82d4a6210f8f"

	input := replicate.PredictionInput{
		"prompt": "An astronaut riding a rainbow unicorn",
	}

	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}

	// The `Run` method is a convenience method that
	// creates a prediction, waits for it to finish, and returns the output.
	// If you want a reference to the prediction, you can call `CreatePrediction`,
	// call `Wait` on the prediction, and access its `Output` field.
	prediction, err := r8.CreatePrediction(ctx, version, input, &webhook, false)
	if err != nil {
		panic(err)
	}

	// Wait for the prediction to finish
	err = r8.Wait(ctx, prediction)
	if err != nil {
		panic(err)
	}
	fmt.Println(prediction.Status)
	// Output: succeeded
}
