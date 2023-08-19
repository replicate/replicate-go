package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/replicate/replicate-go"
)

func main() {
	ngrokURL, ok := os.LookupEnv("REPLICATE_NGROK_URL")
	if !ok || len(ngrokURL) == 0 {
		return
	}

	client, err := replicate.NewClient(replicate.WithTokenFromEnv())
	if err != nil {
		log.Fatal(err)
	}

	// https://replicate.com/stability-ai/stable-diffusion
	version := "stability-ai/stable-diffusion:ac732df83cea7fff18b8472768c88ad041fa750ff7682a21affe81863cbe77e4"

	input := replicate.PredictionInput{
		"prompt": "an astronaut riding a horse on mars, hd, dramatic lighting",
	}
	webhook := replicate.Webhook{
		URL:    fmt.Sprint(ngrokURL, "/webhook"),
		Events: []replicate.WebhookEventType{"completed"},
	}

	if _, err := client.Run(context.Background(), version, input, &webhook); err != nil {
		log.Fatal(err)
	}

}
