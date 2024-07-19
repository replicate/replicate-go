package replicate_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/replicate/replicate-go"
)

func ExampleClient_Run() {
	ctx := context.TODO()

	r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
	if err != nil {
		panic(err)
	}

	model := "bytedance/sdxl-lightning-4step"
	version := "5f24084160c9089501c1b3545d9be3c27883ae2239b6f412990e82d4a6210f8f"

	input := replicate.PredictionInput{
		"prompt": "An astronaut riding a rainbow unicorn",
	}

	output, err := r8.Run(ctx, fmt.Sprintf("%s:%s", model, version), input, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Generated %d image(s)\n", len(output.([]any)))
	// Output: Generated 1 image(s)
}

func ExampleClient_CreatePrediction() {
	ctx := context.TODO()

	r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
	if err != nil {
		panic(err)
	}

	// https://replicate.com/bytedance/sdxl-lightning-4step
	version := "5f24084160c9089501c1b3545d9be3c27883ae2239b6f412990e82d4a6210f8f"

	input := replicate.PredictionInput{
		"prompt": "An astronaut riding a rainbow unicorn",
	}

	prediction, err := r8.CreatePrediction(ctx, version, input, nil, false)
	if err != nil {
		panic(err)
	}

	err = r8.Wait(ctx, prediction)
	if err != nil {
		panic(err)
	}

	fmt.Println(prediction.Status)
	// Output: succeeded
}

func ExampleClient_SearchModels() {
	ctx := context.TODO()

	r8, err := replicate.NewClient(replicate.WithTokenFromEnv())
	if err != nil {
		panic(err)
	}

	query := "llama"
	modelsPage, err := r8.SearchModels(ctx, query)
	if err != nil {
		panic(err)
	}

	for _, model := range modelsPage.Results {
		if model.Owner == "meta" && strings.HasPrefix(model.Name, "meta-llama-3") {
			fmt.Printf("Found Meta Llama 3 model")
			break
		}
	}
	// Output: Found Meta Llama 3 model
}
