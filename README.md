# Replicate Go client

A Go client for [Replicate](https://replicate.com).
It lets you run models from your Go code,
and everything else you can do with
[Replicate's HTTP API](https://replicate.com/docs/reference/http).

## Requirements

- Go 1.20+

## Installation

Use `go get` to install the Replicate package:

```console
go get -u github.com/replicate/replicate-go/replicate
```

Include the Replicate package in your project:

```go
import "github.com/replicate/replicate-go/replicate"
```

## Usage

```go
import (
	"context"
	"os"

	"github.com/replicate/replicate-go/replicate"
)

client := replicate.NewClient(os.Getenv("REPLICATE_API_TOKEN"))

// https://replicate.com/stability-ai/stable-diffusion
version := "db21e45d3f7023abc2a46ee38a23973f6dce16bb082a930b0c49861f96d1e5bf"

input := replicate.PredictionInput{
    "prompt": "an astronaut riding a horse on mars, hd, dramatic lighting",
}

webhook := replicate.Webhook{
  	URL:    "https://example.com/webhook",
  	Events: []replicate.WebhookEventType{"start", "completed"},
}

prediction, err := client.CreatePrediction(context.Background(), version, input, &webhook)
```

## License

Replicate's Go client is released under the Apache 2.0 license.
See [LICENSE.txt](LICENSE.txt)
