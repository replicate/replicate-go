package replicate_test

import (
	"bytes"
	"context"
	"crypto/md5" // nolint:gosec
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/replicate/replicate-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientNoAuth(t *testing.T) {
	_, err := replicate.NewClient()

	assert.ErrorIs(t, err, replicate.ErrNoAuth)
}

func TestNewClientBlankAuthTokenFromEnv(t *testing.T) {
	t.Setenv("REPLICATE_API_TOKEN", "")
	_, err := replicate.NewClient(replicate.WithTokenFromEnv())
	require.ErrorContains(t, err, "REPLICATE_API_TOKEN")
}

func TestNewClientAuthTokenFromEnv(t *testing.T) {
	t.Setenv("REPLICATE_API_TOKEN", "test-token")
	_, err := replicate.NewClient(replicate.WithTokenFromEnv())
	require.NoError(t, err)
}

func TestListCollections(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/collections", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		var response replicate.Page[replicate.Collection]

		mockCursor := "cD0yMDIyLTAxLTIxKzIzJTNBMTglM0EyNC41MzAzNTclMkIwMCUzQTAw"

		switch r.URL.Query().Get("cursor") {
		case "":
			next := "/collections?cursor=" + mockCursor
			response = replicate.Page[replicate.Collection]{
				Previous: nil,
				Next:     &next,
				Results: []replicate.Collection{
					{Slug: "collection-1", Name: "Collection 1", Description: ""},
				},
			}
		case mockCursor:
			previous := "/collections?cursor=" + mockCursor
			response = replicate.Page[replicate.Collection]{
				Previous: &previous,
				Next:     nil,
				Results: []replicate.Collection{
					{Slug: "collection-2", Name: "Collection 2", Description: ""},
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

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initialPage, err := client.ListCollections(ctx)
	if err != nil {
		t.Fatal(err)
	}

	resultsChan, errChan := replicate.Paginate(ctx, client, initialPage)

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

	assert.Equal(t, 2, len(collections))

	assert.Equal(t, "collection-1", collections[0].Slug)
	assert.Equal(t, "Collection 1", collections[0].Name)

	assert.Equal(t, "collection-2", collections[1].Slug)
	assert.Equal(t, "Collection 2", collections[1].Name)
}

func TestGetCollection(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
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

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection, err := client.GetCollection(ctx, "super-resolution")
	assert.NoError(t, err)
	assert.Equal(t, "Super resolution", collection.Name)
	assert.Equal(t, "super-resolution", collection.Slug)
	assert.Equal(t, "Upscaling models that create high-quality images from low-quality images.", collection.Description)
	assert.Empty(t, *collection.Models)
}

func TestListModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := replicate.Page[replicate.Model]{
			Results: []replicate.Model{
				{
					Owner:       "stability-ai",
					Name:        "sdxl",
					Description: "A text-to-image generative AI model that creates beautiful 1024x1024 images",
				},
				{
					Owner:       "meta",
					Name:        "codellama-13b",
					Description: "A 13 billion parameter Llama tuned for code completion",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(response)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	modelsPage, err := client.ListModels(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(modelsPage.Results))
	assert.Equal(t, "stability-ai", modelsPage.Results[0].Owner)
	assert.Equal(t, "sdxl", modelsPage.Results[0].Name)
	assert.Equal(t, "meta", modelsPage.Results[1].Owner)
	assert.Equal(t, "codellama-13b", modelsPage.Results[1].Name)
}

func TestGetModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models/replicate/hello-world", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		model := replicate.Model{
			Owner:          "replicate",
			Name:           "hello-world",
			Description:    "A tiny model that says hello",
			Visibility:     "public",
			GithubURL:      "https://github.com/replicate/cog-examples",
			PaperURL:       "",
			LicenseURL:     "",
			RunCount:       12345,
			CoverImageURL:  "",
			DefaultExample: nil,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(model)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model, err := client.GetModel(ctx, "replicate", "hello-world")
	assert.NoError(t, err)
	assert.Equal(t, "replicate", model.Owner)
	assert.Equal(t, "hello-world", model.Name)
}

func TestCreateModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/models", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		var requestBody map[string]interface{}
		err = json.Unmarshal(body, &requestBody)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "owner", requestBody["owner"])
		assert.Equal(t, "name", requestBody["name"])
		assert.Equal(t, "public", requestBody["visibility"])
		assert.Equal(t, "cpu", requestBody["hardware"])

		response := replicate.Model{
			Owner:          "owner",
			Name:           "name",
			Description:    "",
			Visibility:     "public",
			GithubURL:      "",
			PaperURL:       "",
			LicenseURL:     "",
			RunCount:       0,
			CoverImageURL:  "",
			DefaultExample: nil,
			LatestVersion:  nil,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := replicate.CreateModelOptions{
		Visibility: "public",
		Hardware:   "cpu",
	}
	model, err := client.CreateModel(ctx, "owner", "name", options)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "owner", model.Owner)
	assert.Equal(t, "name", model.Name)
	assert.Equal(t, "public", model.Visibility)
	assert.Equal(t, "", model.Description)
}

func TestDeleteModelVersion(t *testing.T) {
	modelName := "replicate"
	modelOwner := "hello-world"
	versionID := "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, fmt.Sprintf("/models/%s/%s/versions/%s", modelOwner, modelName, versionID), r.URL.Path)
		w.WriteHeader(http.StatusAccepted)

	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.DeleteModelVersion(ctx, modelOwner, modelName, versionID)
	assert.NoError(t, err)
}

func TestListModelVersions(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models/replicate/hello-world/versions", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		versionsPage := replicate.Page[replicate.ModelVersion]{
			Results: []replicate.ModelVersion{
				{ID: "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532"},
				{ID: "b21cbe271e65c1718f2999b038c18b45e21e4fba961181fbfae9342fc53b9e05"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(versionsPage)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	versionsPage, err := client.ListModelVersions(ctx, "replicate", "hello-world")
	assert.NoError(t, err)
	assert.Equal(t, "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532", versionsPage.Results[0].ID)
	assert.Equal(t, "b21cbe271e65c1718f2999b038c18b45e21e4fba961181fbfae9342fc53b9e05", versionsPage.Results[1].ID)
}

func TestGetModelVersion(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/models/replicate/hello-world/versions/version1", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		version := replicate.ModelVersion{
			ID:            "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			CreatedAt:     "2022-04-26T19:29:04.418669Z",
			CogVersion:    "0.3.0",
			OpenAPISchema: map[string]interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(version)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := client.GetModelVersion(ctx, "replicate", "hello-world", "version1")
	assert.NoError(t, err)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", version.ID)
}

func TestCreatePrediction(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/predictions", r.URL.Path)

		body, err := io.ReadAll(r.Body)
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
		assert.Equal(t, true, requestBody["stream"])

		response := replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Model:     "replicate/hello-world",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    "starting",
			Input:     map[string]interface{}{"text": "Alice"},
			Output:    nil,
			Error:     nil,
			Logs:      nil,
			Metrics:   nil,
			CreatedAt: "2022-04-26T22:13:06.224088Z",
			URLs: map[string]string{
				"get":    "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
				"cancel": "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel",
				"stream": "https://streaming.api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
			},
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := replicate.PredictionInput{"text": "Alice"}
	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}
	version := "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa"
	prediction, err := client.CreatePrediction(ctx, version, input, &webhook, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", prediction.Version)
	assert.Equal(t, replicate.Starting, prediction.Status)
	assert.Equal(t, "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq", prediction.URLs["get"])
	assert.Equal(t, "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel", prediction.URLs["cancel"])
	assert.Equal(t, "https://streaming.api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq", prediction.URLs["stream"])
}

func TestCreatePredictionWithDeployment(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/deployments/owner/name/predictions", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		var requestBody map[string]interface{}
		err = json.Unmarshal(body, &requestBody)
		if err != nil {
			t.Fatal(err)
		}

		assert.Nil(t, requestBody["version"])
		assert.Equal(t, map[string]interface{}{"text": "Alice"}, requestBody["input"])
		assert.Equal(t, "https://example.com/webhook", requestBody["webhook"])
		if _, exists := requestBody["webhook_events_filter"]; exists {
			assert.Fail(t, "webhook_events_filter should not be present")
		}
		if _, exists := requestBody["stream"]; exists {
			assert.Fail(t, "stream should not be present")
		}

		response := replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Model:     "replicate/hello-world",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    "starting",
			Input:     map[string]interface{}{"text": "Alice"},
			Output:    nil,
			Error:     nil,
			Logs:      nil,
			Metrics:   nil,
			CreatedAt: "2022-04-26T22:13:06.224088Z",
			URLs: map[string]string{
				"get":    "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
				"cancel": "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel",
				"stream": "https://streaming.api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
			},
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := replicate.PredictionInput{"text": "Alice"}
	webhook := replicate.Webhook{
		URL: "https://example.com/webhook",
	}
	prediction, err := client.CreatePredictionWithDeployment(ctx, "owner", "name", input, &webhook, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", prediction.Version)
	assert.Equal(t, replicate.Starting, prediction.Status)
	assert.Equal(t, "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq", prediction.URLs["get"])
	assert.Equal(t, "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel", prediction.URLs["cancel"])
	assert.Equal(t, "https://streaming.api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq", prediction.URLs["stream"])
}

func TestCreatePredictionWithModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/models/owner/model/predictions", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		var requestBody map[string]interface{}
		err = json.Unmarshal(body, &requestBody)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, map[string]interface{}{"text": "Alice"}, requestBody["input"])
		assert.Equal(t, "https://example.com/webhook", requestBody["webhook"])
		assert.Equal(t, []interface{}{"start", "completed"}, requestBody["webhook_events_filter"])
		assert.Equal(t, true, requestBody["stream"])

		response := replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Model:     "owner/model",
			Status:    "starting",
			Input:     map[string]interface{}{"text": "Alice"},
			Output:    nil,
			Error:     nil,
			Logs:      nil,
			Metrics:   nil,
			CreatedAt: "2022-04-26T22:13:06.224088Z",
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := replicate.PredictionInput{"text": "Alice"}
	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}
	prediction, err := client.CreatePredictionWithModel(ctx, "owner", "model", input, &webhook, true)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, "owner/model", prediction.Model)
	assert.Equal(t, replicate.Starting, prediction.Status)
}

func TestCancelPrediction(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq/cancel", r.URL.Path)

		response := replicate.Prediction{
			ID:     "ufawqhfynnddngldkgtslldrkq",
			Status: replicate.Canceled,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prediction, err := client.CancelPrediction(ctx, "ufawqhfynnddngldkgtslldrkq")
	assert.NoError(t, err)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, replicate.Canceled, prediction.Status)
}

func TestPredictionProgress(t *testing.T) {
	prediction := replicate.Prediction{
		ID:        "ufawqhfynnddngldkgtslldrkq",
		Model:     "replicate/hello-world",
		Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
		Status:    "starting",
		Input:     map[string]interface{}{"text": "Alice"},
		Output:    nil,
		Error:     nil,
		Logs:      nil,
		Metrics:   nil,
		CreatedAt: "2022-04-26T22:13:06.224088Z",
		URLs: map[string]string{
			"get":    "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
			"cancel": "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel",
			"stream": "https://streaming.api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
		},
	}

	lines := []string{
		"Using seed: 12345",
		"0%|          | 0/5 [00:00<?, ?it/s]",
		"20%|██        | 1/5 [00:00<00:01, 21.38it/s]",
		"40%|████▍     | 2/5 [00:01<00:01, 22.46it/s]",
		"60%|████▍     | 3/5 [00:01<00:01, 22.46it/s]",
		"80%|████████  | 4/5 [00:01<00:00, 22.86it/s]",
		"100%|██████████| 5/5 [00:02<00:00, 22.26it/s]",
	}
	logs := ""

	for i, line := range lines {
		logs = logs + "\n" + line
		prediction.Logs = &logs

		progress := prediction.Progress()

		switch i {
		case 0:
			prediction.Status = replicate.Processing
			assert.Nil(t, progress)
		case 1:
			assert.NotNil(t, progress)
			assert.Equal(t, 0, progress.Current)
			assert.Equal(t, 5, progress.Total)
			assert.Equal(t, 0.0, progress.Percentage)
		case 2:
			assert.NotNil(t, progress)
			assert.Equal(t, 1, progress.Current)
			assert.Equal(t, 5, progress.Total)
			assert.Equal(t, 0.2, progress.Percentage)
		case 3:
			assert.NotNil(t, progress)
			assert.Equal(t, 2, progress.Current)
			assert.Equal(t, 5, progress.Total)
			assert.Equal(t, 0.4, progress.Percentage)
		case 4:
			assert.NotNil(t, progress)
			assert.Equal(t, 3, progress.Current)
			assert.Equal(t, 5, progress.Total)
			assert.Equal(t, 0.6, progress.Percentage)
		case 5:
			assert.NotNil(t, progress)
			assert.Equal(t, 4, progress.Current)
			assert.Equal(t, 5, progress.Total)
			assert.Equal(t, 0.8, progress.Percentage)
		case 6:
			assert.NotNil(t, progress)
			prediction.Status = replicate.Succeeded
			assert.Equal(t, 5, progress.Current)
			assert.Equal(t, 5, progress.Total)
			assert.Equal(t, 1.0, progress.Percentage)
		}
	}
}

func TestListPredictions(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/predictions", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		var response replicate.Page[replicate.Prediction]

		mockCursor := "cD0yMDIyLTAxLTIxKzIzJTNBMTglM0EyNC41MzAzNTclMkIwMCUzQTAw"

		switch r.URL.Query().Get("cursor") {
		case "":
			next := "/predictions?cursor=" + mockCursor
			response = replicate.Page[replicate.Prediction]{
				Previous: nil,
				Next:     &next,
				Results: []replicate.Prediction{
					{ID: "ufawqhfynnddngldkgtslldrkq"},
				},
			}
		case mockCursor:
			previous := "/predictions?cursor=" + mockCursor
			response = replicate.Page[replicate.Prediction]{
				Previous: &previous,
				Next:     nil,
				Results: []replicate.Prediction{
					{ID: "rrr4z55ocneqzikepnug6xezpe"},
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

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initialPage, err := client.ListPredictions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	resultsChan, errChan := replicate.Paginate(ctx, client, initialPage)

	var predictions []replicate.Prediction
	for results := range resultsChan {
		predictions = append(predictions, results...)
	}

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	default:
	}

	assert.Len(t, predictions, 2)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", predictions[0].ID)
	assert.Equal(t, "rrr4z55ocneqzikepnug6xezpe", predictions[1].ID)
}

func TestGetPrediction(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Model:     "replicate/hello-world",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    replicate.Succeeded,
			Input:     replicate.PredictionInput{"text": "Alice"},
			Output:    map[string]interface{}{"text": "Hello, Alice"},
			CreatedAt: "2022-04-26T22:13:06.224088Z",
			URLs: map[string]string{
				"get":    "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
				"cancel": "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(prediction)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prediction, err := client.GetPrediction(ctx, "ufawqhfynnddngldkgtslldrkq")
	assert.NoError(t, err)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, replicate.Succeeded, prediction.Status)
	assert.Equal(t, replicate.PredictionInput{"text": "Alice"}, prediction.Input)
	assert.Equal(t, map[string]interface{}{"text": "Hello, Alice"}, prediction.Output)
	assert.Equal(t, "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq", prediction.URLs["get"])
	assert.Equal(t, "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel", prediction.URLs["cancel"])
}

func TestWait(t *testing.T) {
	statuses := []replicate.Status{replicate.Starting, replicate.Processing, replicate.Succeeded}

	i := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Model:     "replicate/hello-world",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    statuses[i],
			Input:     replicate.PredictionInput{"text": "Alice"},
			CreatedAt: "2022-04-26T22:13:06.224088Z",
		}

		if statuses[i] == replicate.Succeeded {
			prediction.Output = map[string]interface{}{"text": "Hello, Alice"}

			startedAt := "2022-04-26T22:13:06.324088Z"
			prediction.StartedAt = &startedAt

			completedAt := "2022-04-26T22:13:07.224088Z"
			prediction.CompletedAt = &completedAt

			predictTime := 0.5
			totalTime := 1.0
			inputTokenCount := 1
			outputTokenCount := 2
			prediction.Metrics = &replicate.PredictionMetrics{
				PredictTime:      &predictTime,
				TotalTime:        &totalTime,
				InputTokenCount:  &inputTokenCount,
				OutputTokenCount: &outputTokenCount,
			}
		}

		if i < len(statuses)-1 {
			i++
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(prediction)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	prediction := &replicate.Prediction{
		ID:        "ufawqhfynnddngldkgtslldrkq",
		Model:     "replicate/hello-world",
		Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
		Status:    replicate.Starting,
		Input:     replicate.PredictionInput{"text": "Alice"},
		CreatedAt: "2022-04-26T22:13:06.224088Z",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Wait(ctx, prediction, replicate.WithPollingInterval(1*time.Nanosecond))
	assert.NoError(t, err)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
	assert.Equal(t, replicate.Succeeded, prediction.Status)
	assert.Equal(t, replicate.PredictionInput{"text": "Alice"}, prediction.Input)
	assert.Equal(t, map[string]interface{}{"text": "Hello, Alice"}, prediction.Output)
	assert.Equal(t, "2022-04-26T22:13:06.324088Z", *prediction.StartedAt)
	assert.Equal(t, "2022-04-26T22:13:07.224088Z", *prediction.CompletedAt)
	assert.Equal(t, 0.5, *prediction.Metrics.PredictTime)
	assert.Equal(t, 1.0, *prediction.Metrics.TotalTime)
	assert.Equal(t, 1, *prediction.Metrics.InputTokenCount)
	assert.Equal(t, 2, *prediction.Metrics.OutputTokenCount)
}

func TestWaitAsync(t *testing.T) {
	statuses := []replicate.Status{replicate.Starting, replicate.Processing, replicate.Succeeded}

	i := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
			Model:     "replicate/hello-world",
			Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
			Status:    statuses[i],
			Input:     replicate.PredictionInput{"text": "Alice"},
			CreatedAt: "2022-04-26T22:13:06.224088Z",
		}

		if statuses[i] == replicate.Succeeded {
			prediction.Output = map[string]interface{}{"text": "Hello, Alice"}
		}

		if i < len(statuses)-1 {
			i++
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(prediction)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	prediction := &replicate.Prediction{
		ID:        "ufawqhfynnddngldkgtslldrkq",
		Model:     "replicate/hello-world",
		Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
		Status:    replicate.Starting,
		Input:     replicate.PredictionInput{"text": "Alice"},
		CreatedAt: "2022-04-26T22:13:06.224088Z",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	predChan, errChan := client.WaitAsync(ctx, prediction, replicate.WithPollingInterval(1*time.Nanosecond))
	var lastStatus replicate.Status
	for pred := range predChan {
		lastStatus = pred.Status
		if pred.Status == replicate.Succeeded {
			assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", pred.ID)
			assert.Equal(t, replicate.PredictionInput{"text": "Alice"}, pred.Input)
			assert.Equal(t, map[string]interface{}{"text": "Hello, Alice"}, pred.Output)
			break
		}
	}

	if err := <-errChan; err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, replicate.Succeeded, lastStatus)
}

func TestCreateTraining(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/models/owner/model/versions/632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532/trainings", r.URL.Path)

		training := &replicate.Training{
			ID:        "zz4ibbonubfz7carwiefibzgga",
			Model:     "replicate/hello-world",
			Version:   "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532",
			Status:    replicate.Starting,
			CreatedAt: "2023-03-28T21:47:58.566434Z",
			URLs: map[string]string{
				"get":    "https://api.replicate.com/v1/trainings/zz4ibbonubfz7carwiefibzgga",
				"cancel": "https://api.replicate.com/v1/trainings/zz4ibbonubfz7carwiefibzgga/cancel",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		body, _ := json.Marshal(training)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := replicate.TrainingInput{"text": "Alice"}
	destination := "owner/new-model"
	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}
	training, err := client.CreateTraining(ctx, "owner", "model", "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532", destination, input, &webhook)
	if err != nil {
		t.Fatal(err)
	}

	assert.NoError(t, err)
	assert.Equal(t, "zz4ibbonubfz7carwiefibzgga", training.ID)
	assert.Equal(t, "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532", training.Version)
	assert.Equal(t, replicate.Starting, training.Status)
	assert.Equal(t, "https://api.replicate.com/v1/trainings/zz4ibbonubfz7carwiefibzgga", training.URLs["get"])
	assert.Equal(t, "https://api.replicate.com/v1/trainings/zz4ibbonubfz7carwiefibzgga/cancel", training.URLs["cancel"])
}

func TestGetTraining(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trainings/zz4ibbonubfz7carwiefibzgga", r.URL.Path)

		training := &replicate.Training{
			ID:        "zz4ibbonubfz7carwiefibzgga",
			Model:     "replicate/hello-world",
			Version:   "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532",
			Status:    replicate.Succeeded,
			CreatedAt: "2023-03-28T21:47:58.566434Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(training)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	training, err := client.GetTraining(ctx, "zz4ibbonubfz7carwiefibzgga")
	assert.NoError(t, err)
	assert.Equal(t, "zz4ibbonubfz7carwiefibzgga", training.ID)
	assert.Equal(t, "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532", training.Version)
	assert.Equal(t, replicate.Succeeded, training.Status)
}

func TestCancelTraining(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trainings/zz4ibbonubfz7carwiefibzgga/cancel", r.URL.Path)

		training := &replicate.Training{
			ID:        "zz4ibbonubfz7carwiefibzgga",
			Model:     "replicate/hello-world",
			Version:   "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532",
			Status:    replicate.Canceled,
			CreatedAt: "2023-03-28T21:47:58.566434Z",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(training)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	training, err := client.CancelTraining(ctx, "zz4ibbonubfz7carwiefibzgga")
	assert.NoError(t, err)
	assert.Equal(t, "zz4ibbonubfz7carwiefibzgga", training.ID)
	assert.Equal(t, "632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532", training.Version)
	assert.Equal(t, replicate.Canceled, training.Status)
}

func TestListTrainings(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trainings", r.URL.Path)

		response := &replicate.Page[replicate.Training]{
			Previous: nil,
			Next:     nil,
			Results: []replicate.Training{
				{ID: "ufawqhfynnddngldkgtslldrkq"},
				{ID: "rrr4z55ocneqzikepnug6xezpe"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(response)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	page, err := client.ListTrainings(ctx)
	assert.NoError(t, err)
	assert.Len(t, page.Results, 2)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", page.Results[0].ID)
	assert.Equal(t, "rrr4z55ocneqzikepnug6xezpe", page.Results[1].ID)
}

func TestListHardware(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/hardware", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := []replicate.Hardware{
			{Name: "CPU", SKU: "cpu"},
			{Name: "Nvidia T4 GPU", SKU: "gpu-t4"},
			{Name: "Nvidia A40 GPU", SKU: "gpu-a40-small"},
			{Name: "Nvidia A40 (Large) GPU", SKU: "gpu-a40-large"},
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hardwareList, err := client.ListHardware(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 4, len(*hardwareList))

	assert.Equal(t, "CPU", (*hardwareList)[0].Name)
	assert.Equal(t, "cpu", (*hardwareList)[0].SKU)
}

func TestAutomaticallyRetryGetRequests(t *testing.T) {
	statuses := []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusOK}

	i := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		status := statuses[i]
		i++

		if status == http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)

			prediction := &replicate.Prediction{
				ID:        "ufawqhfynnddngldkgtslldrkq",
				Model:     "replicate/hello-world",
				Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
				Status:    replicate.Succeeded,
				Input:     replicate.PredictionInput{"text": "Alice"},
				Output:    map[string]interface{}{"text": "Hello, Alice"},
				CreatedAt: "2022-04-26T22:13:06.224088Z",
				URLs: map[string]string{
					"get":    "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq",
					"cancel": "https://api.replicate.com/v1/predictions/ufawqhfynnddngldkgtslldrkq/cancel",
				},
			}

			body, _ := json.Marshal(prediction)
			w.Write(body)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(status)

			err := replicate.APIError{
				Detail: http.StatusText(status),
			}
			body, _ := json.Marshal(err)
			w.Write(body)
		}
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prediction, err := client.GetPrediction(ctx, "ufawqhfynnddngldkgtslldrkq")
	assert.NoError(t, err)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", prediction.ID)
}

func TestAutomaticallyRetryPostRequests(t *testing.T) {
	statuses := []int{http.StatusTooManyRequests, http.StatusInternalServerError}

	i := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		status := statuses[i]
		i++

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(status)

		err := replicate.APIError{
			Detail: http.StatusText(status),
		}
		body, _ := json.Marshal(err)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := replicate.PredictionInput{"text": "Alice"}
	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}
	version := "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa"
	_, err = client.CreatePrediction(ctx, version, input, &webhook, true)

	assert.ErrorContains(t, err, http.StatusText(http.StatusInternalServerError))
}

func TestStream(t *testing.T) {
	tokens := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"}

	mockServer := httptest.NewUnstartedServer(nil)
	mockServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/predictions":
			body, err := io.ReadAll(r.Body)
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
			assert.Equal(t, true, requestBody["stream"])

			response := replicate.Prediction{
				ID:        "ufawqhfynnddngldkgtslldrkq",
				Model:     "replicate/hello-world",
				Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
				Status:    "starting",
				Input:     map[string]interface{}{"text": "Alice"},
				CreatedAt: "2022-04-26T22:13:06.224088Z",
				URLs: map[string]string{
					"stream": fmt.Sprintf("%s/predictions/ufawqhfynnddngldkgtslldrkq/stream", mockServer.URL),
				},
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}

			w.WriteHeader(http.StatusCreated)
			w.Write(responseBytes)
		case r.Method == http.MethodGet && r.URL.Path == "/predictions/ufawqhfynnddngldkgtslldrkq/stream":
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			for _, token := range tokens {
				fmt.Fprintf(w, "data: %s\n\n", token)
				flusher.Flush()
				time.Sleep(time.Millisecond * 10)
			}
		default:
			t.Fatalf("Unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	mockServer.Start()
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := replicate.PredictionInput{"text": "Alice"}
	version := "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa"

	sseChan, errChan := client.Stream(ctx, fmt.Sprintf("replicate/hello-world:%s", version), input, nil)

	for _, token := range tokens {
		select {
		case <-time.After(10 * time.Second):
			t.Fatal("timeout")
		case event := <-sseChan:
			assert.Equal(t, token, event.Data)
		case err := <-errChan:
			assert.NoError(t, err)
		}
	}
}

func TestCreateFile(t *testing.T) {
	fileID := "file-id"
	options := &replicate.CreateFileOptions{
		Filename:    "hello.txt",
		ContentType: "text/plain",
		Metadata:    map[string]string{"foo": "bar"},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/files", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		defer r.Body.Close()

		part, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "form-data; name=\"content\"; filename=\"hello.txt\"", part.Header.Get("Content-Disposition"))
		assert.Equal(t, "text/plain", part.Header.Get("Content-Type"))

		content, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}

		etag := fmt.Sprintf("%x", md5.Sum(content)) // nolint:gosec
		checksum := sha256.Sum256(content)
		file := &replicate.File{
			ID:          fileID,
			Name:        "hello.txt",
			ContentType: "text/plain",
			Size:        len(content),
			Etag:        etag,
			Checksums:   map[string]string{"sha256": hex.EncodeToString(checksum[:])},
			Metadata:    map[string]string{"foo": "bar"},
			CreatedAt:   "2022-04-26T22:13:06.224088Z",
			URLs:        map[string]string{"get": "https://api.replicate.com/v1/files/" + fileID},
		}

		responseBytes, err := json.Marshal(file)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("CreateFileFromBytes", func(t *testing.T) {
		content := []byte("Hello, world!")
		file, err := client.CreateFileFromBytes(ctx, content, options)
		if err != nil {
			t.Fatal(err)
		}
		assertCreatedFile(t, fileID, file)
	})

	t.Run("CreateFileFromBuffer", func(t *testing.T) {
		buf := bytes.NewBufferString("Hello, world!")
		file, err := client.CreateFileFromBuffer(ctx, buf, options)
		if err != nil {
			t.Fatal(err)
		}
		assertCreatedFile(t, fileID, file)
	})

	t.Run("CreateFileFromPath", func(t *testing.T) {
		content := []byte("Hello, world!")
		tmpFilePath := filepath.Join(t.TempDir(), "hello.txt")
		if err := os.WriteFile(tmpFilePath, content, 0o644); err != nil {
			t.Fatal(err)
		}
		file, err := client.CreateFileFromPath(ctx, tmpFilePath, options)
		if err != nil {
			t.Fatal(err)
		}
		assertCreatedFile(t, fileID, file)
	})
}

func assertCreatedFile(t *testing.T, fileID string, file *replicate.File) {
	assert.Equal(t, fileID, file.ID)
	assert.Equal(t, "hello.txt", file.Name)
	assert.Equal(t, "text/plain", file.ContentType)
	assert.Equal(t, 13, file.Size)
	assert.Equal(t, "6cd3556deb0da54bca060b4c39479839", file.Etag)
	assert.Equal(t, "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3", file.Checksums["sha256"])
	assert.Equal(t, map[string]string{"foo": "bar"}, file.Metadata)
	assert.Equal(t, "2022-04-26T22:13:06.224088Z", file.CreatedAt)
	assert.Equal(t, "https://api.replicate.com/v1/files/"+fileID, file.URLs["get"])
}

func TestListFiles(t *testing.T) {
	fileID := "file-id"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/files", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := replicate.Page[replicate.File]{
			Results: []replicate.File{
				{
					ID:          fileID,
					Name:        "hello.txt",
					ContentType: "text/plain",
					Size:        13,
					CreatedAt:   "2022-04-26T22:13:06.224088Z",
					URLs:        map[string]string{"get": "https://api.replicate.com/v1/files/" + fileID},
				},
			},
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	files, err := client.ListFiles(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(files.Results))
	assert.Nil(t, files.Previous)
	assert.Nil(t, files.Next)

	file := files.Results[0]
	assert.Equal(t, fileID, file.ID)
	assert.Equal(t, "hello.txt", file.Name)
	assert.Equal(t, "text/plain", file.ContentType)
	assert.Equal(t, 13, file.Size)
	assert.Equal(t, "2022-04-26T22:13:06.224088Z", file.CreatedAt)
	assert.Equal(t, "https://api.replicate.com/v1/files/"+fileID, file.URLs["get"])
}
func TestGetFile(t *testing.T) {
	fileID := "file-id"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/files/"+fileID, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		file := &replicate.File{
			ID:          fileID,
			Name:        "hello.txt",
			ContentType: "text/plain",
			Size:        13,
			CreatedAt:   "2022-04-26T22:13:06.224088Z",
			URLs:        map[string]string{"get": "https://api.replicate.com/v1/files/" + fileID},
		}

		responseBytes, err := json.Marshal(file)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	file, err := client.GetFile(ctx, fileID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, fileID, file.ID)
	assert.Equal(t, "hello.txt", file.Name)
	assert.Equal(t, "text/plain", file.ContentType)
	assert.Equal(t, 13, file.Size)
	assert.Equal(t, "2022-04-26T22:13:06.224088Z", file.CreatedAt)
	assert.Equal(t, "https://api.replicate.com/v1/files/"+fileID, file.URLs["get"])
}

func TestDeleteFile(t *testing.T) {
	fileID := "file-id"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/files/"+fileID, r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.DeleteFile(ctx, fileID)
	assert.NoError(t, err)
}

func TestGetCurrentAccount(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/account", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		account := replicate.Account{
			Type:      "organization",
			Username:  "replicate",
			Name:      "Replicate",
			GithubURL: "https://github.com/replicate",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(account)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	account, err := client.GetCurrentAccount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "organization", account.Type)
	assert.Equal(t, "replicate", account.Username)
	assert.Equal(t, "Replicate", account.Name)
	assert.Equal(t, "https://github.com/replicate", account.GithubURL)
}

func TestGetDefaultWebhookSecret(t *testing.T) {
	// This is a test secret and should not be used in production
	testSecret := replicate.WebhookSigningSecret{
		Key: "whsec_5WbX5kEWLlfzsGNjH64I8lOOqUB6e8FH", // nolint:gosec
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/webhooks/default/secret", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(testSecret)
		w.Write(body)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secret, err := client.GetDefaultWebhookSecret(ctx)
	assert.NoError(t, err)
	assert.Equal(t, testSecret.Key, secret.Key)
}

func TestValidateWebhook(t *testing.T) {
	// Test case from https://github.com/svix/svix-webhooks/blob/b41728cd98a7e7004a6407a623f43977b82fcba4/javascript/src/webhook.test.ts#L190-L200

	// This is a test secret and should not be used in production
	testSecret := replicate.WebhookSigningSecret{
		Key: "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw", // nolint:gosec
	}

	body := `{"test": 2432232314}`
	req := httptest.NewRequest(http.MethodPost, "http://test.host/webhook", strings.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Webhook-ID", "msg_p5jXN8AQM9LWM0D4loKWxJek")
	req.Header.Add("Webhook-Timestamp", "1614265330")
	req.Header.Add("Webhook-Signature", "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE=")

	isValid, err := replicate.ValidateWebhookRequest(req, testSecret)
	require.NoError(t, err)
	assert.True(t, isValid)
}

func TestGetDeployment(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/deployments/acme/image-upscaler", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		deployment := &replicate.Deployment{
			Owner: "acme",
			Name:  "image-upscaler",
			CurrentRelease: replicate.DeploymentRelease{
				Number:    1,
				Model:     "acme/esrgan",
				Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
				CreatedAt: "2022-01-01T00:00:00Z",
				CreatedBy: replicate.Account{
					Type:     "organization",
					Username: "acme",
					Name:     "Acme, Inc.",
				},
				Configuration: replicate.DeploymentConfiguration{
					Hardware:     "gpu-t4",
					MinInstances: 1,
					MaxInstances: 5,
				},
			},
		}

		responseBytes, err := json.Marshal(deployment)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deployment, err := client.GetDeployment(ctx, "acme", "image-upscaler")
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, deployment)
	assert.Equal(t, "acme", deployment.Owner)
	assert.Equal(t, "image-upscaler", deployment.Name)
	assert.Equal(t, 1, deployment.CurrentRelease.Number)
	assert.Equal(t, "acme/esrgan", deployment.CurrentRelease.Model)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", deployment.CurrentRelease.Version)
	assert.Equal(t, "2022-01-01T00:00:00Z", deployment.CurrentRelease.CreatedAt)
	assert.Equal(t, "organization", deployment.CurrentRelease.CreatedBy.Type)
	assert.Equal(t, "acme", deployment.CurrentRelease.CreatedBy.Username)
	assert.Equal(t, "Acme, Inc.", deployment.CurrentRelease.CreatedBy.Name)
	assert.Equal(t, "gpu-t4", deployment.CurrentRelease.Configuration.Hardware)
	assert.Equal(t, 1, deployment.CurrentRelease.Configuration.MinInstances)
	assert.Equal(t, 5, deployment.CurrentRelease.Configuration.MaxInstances)
}

func TestListDeployments(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/deployments", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		deployments := &replicate.Page[replicate.Deployment]{
			Results: []replicate.Deployment{
				{
					Owner: "acme",
					Name:  "image-upscaler",
					CurrentRelease: replicate.DeploymentRelease{
						Number:    1,
						Model:     "acme/esrgan",
						Version:   "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa",
						CreatedAt: "2022-01-01T00:00:00Z",
						CreatedBy: replicate.Account{
							Type:     "organization",
							Username: "acme",
							Name:     "Acme, Inc.",
						},
						Configuration: replicate.DeploymentConfiguration{
							Hardware:     "gpu-t4",
							MinInstances: 1,
							MaxInstances: 5,
						},
					},
				},
				{
					Owner: "acme",
					Name:  "text-generator",
					CurrentRelease: replicate.DeploymentRelease{
						Number:    2,
						Model:     "acme/acme-llama",
						Version:   "4b7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccbb",
						CreatedAt: "2022-02-02T00:00:00Z",
						CreatedBy: replicate.Account{
							Type:     "organization",
							Username: "acme",
							Name:     "Acme, Inc.",
						},
						Configuration: replicate.DeploymentConfiguration{
							Hardware:     "cpu",
							MinInstances: 2,
							MaxInstances: 10,
						},
					},
				},
			},
		}

		responseBytes, err := json.Marshal(deployments)
		if err != nil {
			t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
	require.NotNil(t, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deployments, err := client.ListDeployments(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, deployments)
	assert.Len(t, deployments.Results, 2)

	// Asserting the first deployment
	assert.Equal(t, "acme", deployments.Results[0].Owner)
	assert.Equal(t, "image-upscaler", deployments.Results[0].Name)
	assert.Equal(t, 1, deployments.Results[0].CurrentRelease.Number)
	assert.Equal(t, "acme/esrgan", deployments.Results[0].CurrentRelease.Model)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", deployments.Results[0].CurrentRelease.Version)
	assert.Equal(t, "2022-01-01T00:00:00Z", deployments.Results[0].CurrentRelease.CreatedAt)
	assert.Equal(t, "organization", deployments.Results[0].CurrentRelease.CreatedBy.Type)
	assert.Equal(t, "acme", deployments.Results[0].CurrentRelease.CreatedBy.Username)
	assert.Equal(t, "Acme, Inc.", deployments.Results[0].CurrentRelease.CreatedBy.Name)
	assert.Equal(t, "gpu-t4", deployments.Results[0].CurrentRelease.Configuration.Hardware)
	assert.Equal(t, 1, deployments.Results[0].CurrentRelease.Configuration.MinInstances)
	assert.Equal(t, 5, deployments.Results[0].CurrentRelease.Configuration.MaxInstances)

	// Asserting the second deployment
	assert.Equal(t, "acme", deployments.Results[1].Owner)
	assert.Equal(t, "text-generator", deployments.Results[1].Name)
	assert.Equal(t, 2, deployments.Results[1].CurrentRelease.Number)
	assert.Equal(t, "acme/acme-llama", deployments.Results[1].CurrentRelease.Model)
	assert.Equal(t, "4b7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccbb", deployments.Results[1].CurrentRelease.Version)
	assert.Equal(t, "2022-02-02T00:00:00Z", deployments.Results[1].CurrentRelease.CreatedAt)
	assert.Equal(t, "organization", deployments.Results[1].CurrentRelease.CreatedBy.Type)
	assert.Equal(t, "acme", deployments.Results[1].CurrentRelease.CreatedBy.Username)
	assert.Equal(t, "Acme, Inc.", deployments.Results[0].CurrentRelease.CreatedBy.Name)
	assert.Equal(t, "cpu", deployments.Results[1].CurrentRelease.Configuration.Hardware)
	assert.Equal(t, 2, deployments.Results[1].CurrentRelease.Configuration.MinInstances)
	assert.Equal(t, 10, deployments.Results[1].CurrentRelease.Configuration.MaxInstances)
}
