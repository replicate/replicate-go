# Replicate Go client

[![Go Reference](https://pkg.go.dev/badge/github.com/replicate/replicate-go.svg)](https://pkg.go.dev/github.com/replicate/replicate-go)

A Go client for [Replicate](https://replicate.com).
It lets you run models from your Go code,
and everything else you can do with
[Replicate's HTTP API](https://replicate.com/docs/reference/http).

## Requirements

- Go 1.20+

## Installation

Use `go get` to install the Replicate package:

```console
go get -u github.com/replicate/replicate-go
```

Include the Replicate package in your project:

```go
import "github.com/replicate/replicate-go"
```

## Usage

```go
import (
	"context"
	"os"

	"github.com/replicate/replicate-go"
)

ctx := context.Background()

// You can also provide a token directly with 
// `replicate.NewClient(replicate.WithToken("r8_..."))`
r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
if err != nil {
	// handle error
}

// https://replicate.com/stability-ai/stable-diffusion
version := "ac732df83cea7fff18b8472768c88ad041fa750ff7682a21affe81863cbe77e4"

input := replicate.PredictionInput{
	"prompt": "an astronaut riding a horse on mars, hd, dramatic lighting",
}

webhook := replicate.Webhook{
	URL:    "https://example.com/webhook",
	Events: []replicate.WebhookEventType{"start", "completed"},
}

// Run a model and wait for its output
output, err := r8.Run(ctx, version, input, &webhook)
if err != nil {
	// handle error
}
fmt.Println("output: ", output)

// Create a prediction
prediction, err := r8.CreatePrediction(ctx, version, input, &webhook, false)
if err != nil {
	// handle error
}

// Wait for the prediction to finish
err = r8.Wait(ctx, prediction)
if err != nil {
	// handle error
}
fmt.Println("output: ", output)


// The `Run` method is a convenience method that
// creates a prediction, waits for it to finish, and returns the output.
// If you want a reference to the prediction, you can call `CreatePrediction`,
// call `Wait` on the prediction, and access its `Output` field.
prediction, err := r8.CreatePrediction(ctx, version, input, &webhook, false)
if err != nil {
	// handle error
}

// Wait for the prediction to finish
err = r8.Wait(ctx, prediction)
if err != nil {
	// handle error
}
fmt.Println("output: ", prediction.Output)
```

## License

Replicate's Go client is released under the Apache 2.0 license.
See [LICENSE.txt](LICENSE.txt)
