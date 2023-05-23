# Replicate Go Client

🚧 WIP 🚧

This is a Go client for [Replicate](https://replicate.com).

## Example Usage

```go
import (
	"os"

	"github.com/replicate/replicate-go/pkg/replicate"
)

client := replicate.New(os.Getenv("REPLICATE_API_TOKEN"))

// https://replicate.com/stability-ai/stable-diffusion
version := "db21e45d3f7023abc2a46ee38a23973f6dce16bb082a930b0c49861f96d1e5bf"
input := replicate.PredictionInput{
    "prompt": "an astronaut riding a horse on mars, hd, dramatic lighting"
}
prediction, err := client.CreatePrediction(context.Background(), version, input)
```
