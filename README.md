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

### Create a client

```go
import (
	"context"
	"os"

	"github.com/replicate/replicate-go"
)

ctx := context.TODO()

// You can also provide a token directly with 
// `replicate.NewClient(replicate.WithToken("r8_..."))`
r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
if err != nil {
	// handle error
}
```

### Run a model

```go
model := "stability-ai/sdxl"
version := "7762fd07cf82c948538e41f63f77d685e02b063e37e496e96eefd46c929f9bdc"

input := replicate.PredictionInput{
	"prompt": "An astronaut riding a rainbow unicorn",
}

webhook := replicate.Webhook{
	URL:    "https://example.com/webhook",
	Events: []replicate.WebhookEventType{"start", "completed"},
}

// Run a model and wait for its output
output, _ := r8.Run(ctx, fmt.Sprintf("%s:%s", model, version), input, &webhook)
```

The `Run` method is a convenience method that
creates a prediction, waits for it to finish, and returns the output.
If you want a reference to the prediction, you can call `CreatePrediction`,
call `Wait` on the prediction, and access its `Output` field.

```go
prediction, _ := r8.CreatePrediction(ctx, version, input, &webhook, false)
_ = r8.Wait(ctx, prediction) // Wait for the prediction to finish
```

Some models take file inputs.
Use the `CreateFileFromPath`, `CreateFileFromBytes`, or `CreateFileFromBuffer` method
to upload a file and pass it as a prediction input.

```go
// https://replicate.com/vaibhavs10/incredibly-fast-whisper
version := "3ab86df6c8f54c11309d4d1f930ac292bad43ace52d10c80d87eb258b3c9f79c"

file, _ := r8.CreateFileFromPath(ctx, "path/to/audio.mp3", nil)

input := replicate.PredictionInput{
	"audio": file,
}
prediction, _ := r8.CreatePrediction(ctx, version, input, nil, false)
```

### Webhooks

To prevent unauthorized requests, Replicate signs every webhook and its metadata with a unique key for each user or organization. You can use this signature to verify the webhook indeed comes from Replicate before you process it.

This client includes a `ValidateWebhookRequest` convenience function that you can use to validate webhooks:

```go
import (
	"github.com/replicate/replicate-go"
)

isValid, err := replicate.ValidateWebhookRequest(req, secret)
```

To learn more, see the [webhooks guide](https://replicate.com/docs/webhooks).

## License

Replicate's Go client is released under the Apache 2.0 license.
See [LICENSE.txt](LICENSE.txt)
