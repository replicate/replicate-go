package replicate_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/replicate/replicate-go/pkg/replicate"
	"github.com/stretchr/testify/assert"
)

func TestListCollections(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/collections", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		type result struct {
			Slug        string `json:"slug"`
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		var response replicate.Page[result]

		mockCursor := "cD0yMDIyLTAxLTIxKzIzJTNBMTglM0EyNC41MzAzNTclMkIwMCUzQTAw"

		switch r.URL.Query().Get("cursor") {
		case "":
			next := "/collections?cursor=" + mockCursor
			response = replicate.Page[result]{
				Previous: nil,
				Next:     &next,
				Results: []result{
					{Slug: "collection-1", Name: "Collection 1", Description: "..."},
				},
			}
		case mockCursor:
			previous := "/collections?cursor=" + mockCursor
			response = replicate.Page[result]{
				Previous: &previous,
				Next:     nil,
				Results: []result{
					{Slug: "collection-2", Name: "Collection 2", Description: "..."},
				},
			}
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client := &replicate.Client{
		BaseURL:    mockServer.URL,
		Auth:       "test-token",
		HTTPClient: http.DefaultClient,
	}

	initialPage, err := client.ListCollections(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	resultsChan, errChan := replicate.Paginate(context.Background(), client, initialPage)

	var collections []replicate.Collection
	for results := range resultsChan {
		collections = append(collections, results...)
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	default:
	}

	expectedCollections := []replicate.Collection{
		{Slug: "collection-1", Name: "Collection 1", Description: "..."},
		{Slug: "collection-2", Name: "Collection 2", Description: "..."},
	}

	assert.Equal(t, expectedCollections, collections)
}

func TestGetCollection(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/collections/super-resolution", r.URL.Path)

		collection := &replicate.Collection{
			Name:        "Super resolution",
			Slug:        "super-resolution",
			Description: "Upscaling models that create high-quality images from low-quality images.",
			Models:      &[]replicate.Model{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(collection)
		w.Write(body)
	}))
	defer mockServer.Close()

	client := &replicate.Client{
		BaseURL:    mockServer.URL,
		Auth:       "test-token",
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection, err := client.GetCollection(ctx, "super-resolution")
	assert.NoError(t, err)
	assert.Equal(t, "Super resolution", collection.Name)
	assert.Equal(t, "super-resolution", collection.Slug)
	assert.Equal(t, "Upscaling models that create high-quality images from low-quality images.", collection.Description)
	assert.Empty(t, *collection.Models)
}

func TestCreatePrediction(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/predictions", r.URL.Path)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		var requestBody map[string]interface{}
		err = json.Unmarshal(body, &requestBody)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", requestBody["version"])
		assert.Equal(t, map[string]interface{}{"text": "Alice"}, requestBody["input"])
		assert.Equal(t, "https://example.com/webhook", requestBody["webhook"])
		assert.Equal(t, []interface{}{"start", "completed"}, requestBody["webhook_events_filter"])

		response := replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    "starting",
			Input:     map[string]interface{}{"text": "Alice"},
			Output:    nil,
			Error:     nil,
			Logs:      nil,
			Metrics:   nil,
			CreatedAt: "2022-04-26T22:13:06.224088Z",
			UpdatedAt: "2022-04-26T22:13:06.224088Z",
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client := &replicate.Client{
		BaseURL:    mockServer.URL,
		Auth:       "test-token",
		HTTPClient: http.DefaultClient,
	}

	input := replicate.PredictionInput{"text": "Alice"}
	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}

	version := "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa"
	prediction, err := client.CreatePrediction(context.Background(), version, input, &webhook)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", prediction.Version)
	assert.Equal(t, replicate.Starting, prediction.Status)
}

func TestGetPrediction(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/predictions/abc123", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "abc123",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    replicate.Succeeded,
			Input:     replicate.PredictionInput{"text": "Alice"},
			Output:    map[string]interface{}{"text": "Hello, Alice"},
			CreatedAt: "2022-04-26T22:13:06.224088Z",
			UpdatedAt: "2022-04-26T22:13:06.224088Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(prediction)
		w.Write(body)
	}))
	defer mockServer.Close()

	client := &replicate.Client{
		BaseURL:    mockServer.URL,
		Auth:       "test-token",
		HTTPClient: http.DefaultClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prediction, err := client.GetPrediction(ctx, "abc123")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", prediction.ID)
	assert.Equal(t, replicate.Succeeded, prediction.Status)
	assert.Equal(t, replicate.PredictionInput{"text": "Alice"}, prediction.Input)
	assert.Equal(t, map[string]interface{}{"text": "Hello, Alice"}, prediction.Output)
}
