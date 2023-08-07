package replicate_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	require.NoError(t, err)

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

	assert.Equal(t, 2, len(collections))

	assert.Equal(t, "collection-1", collections[0].Slug)
	assert.Equal(t, "Collection 1", collections[0].Name)

	assert.Equal(t, "collection-2", collections[1].Slug)
	assert.Equal(t, "Collection 2", collections[1].Name)
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

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
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
	require.NoError(t, err)

	model, err := client.GetModel(context.Background(), "replicate", "hello-world")
	assert.NoError(t, err)
	assert.Equal(t, "replicate", model.Owner)
	assert.Equal(t, "hello-world", model.Name)
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
	require.NoError(t, err)

	versionsPage, err := client.ListModelVersions(context.Background(), "replicate", "hello-world")
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
	require.NoError(t, err)

	version, err := client.GetModelVersion(context.Background(), "replicate", "hello-world", "version1")
	assert.NoError(t, err)
	assert.Equal(t, "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa", version.ID)
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
		assert.Equal(t, true, requestBody["stream"])

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
	require.NoError(t, err)

	input := replicate.PredictionInput{"text": "Alice"}
	webhook := replicate.Webhook{
		URL:    "https://example.com/webhook",
		Events: []replicate.WebhookEventType{"start", "completed"},
	}
	version := "5c7d5dc6dd8bf75c1acaa8565735e7986bc5b66206b55cca93cb72c9bf15ccaa"
	prediction, err := client.CreatePrediction(context.Background(), version, input, &webhook, true)
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
	require.NoError(t, err)

	initialPage, err := client.ListPredictions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	resultsChan, errChan := replicate.Paginate(context.Background(), client, initialPage)

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
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
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
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
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
	require.NoError(t, err)

	prediction := &replicate.Prediction{
		ID:        "ufawqhfynnddngldkgtslldrkq",
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
}

func TestWaitAsync(t *testing.T) {
	statuses := []replicate.Status{replicate.Starting, replicate.Processing, replicate.Succeeded}

	i := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/predictions/ufawqhfynnddngldkgtslldrkq", r.URL.Path)

		prediction := &replicate.Prediction{
			ID:        "ufawqhfynnddngldkgtslldrkq",
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
	require.NoError(t, err)

	prediction := &replicate.Prediction{
		ID:        "ufawqhfynnddngldkgtslldrkq",
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
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/models/owner/model/versions/632231d0d49d34d5c4633bd838aee3d81d936e59a886fbf28524702003b4c532/trainings", r.URL.Path)

		training := &replicate.Training{
			ID:        "zz4ibbonubfz7carwiefibzgga",
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
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/trainings/zz4ibbonubfz7carwiefibzgga", r.URL.Path)

		training := &replicate.Training{
			ID:        "zz4ibbonubfz7carwiefibzgga",
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
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/trainings/zz4ibbonubfz7carwiefibzgga/cancel", r.URL.Path)

		training := &replicate.Training{
			ID:        "zz4ibbonubfz7carwiefibzgga",
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
		assert.Equal(t, "GET", r.Method)
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
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	page, err := client.ListTrainings(ctx)
	assert.NoError(t, err)
	assert.Len(t, page.Results, 2)
	assert.Equal(t, "ufawqhfynnddngldkgtslldrkq", page.Results[0].ID)
	assert.Equal(t, "rrr4z55ocneqzikepnug6xezpe", page.Results[1].ID)
}

func TestAutomaticallyRetryGetRequests(t *testing.T) {
	statuses := []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusOK}

	i := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		status := statuses[i]
		i++

		if status == http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)

			prediction := &replicate.Prediction{
				ID:        "ufawqhfynnddngldkgtslldrkq",
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

			if status == http.StatusInternalServerError {
				err := &replicate.APIError{
					Detail: "Internal server error",
				}
				body, _ := json.Marshal(err)
				w.Write(body)
			} else if status == http.StatusTooManyRequests {
				err := &replicate.APIError{
					Detail: "Too many requests",
				}
				body, _ := json.Marshal(err)
				w.Write(body)
			}
		}
	}))
	defer mockServer.Close()

	client, err := replicate.NewClient(
		replicate.WithToken("test-token"),
		replicate.WithBaseURL(mockServer.URL),
	)
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
		status := statuses[i]
		i++

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(status)

		if status == http.StatusInternalServerError {
			err := &replicate.APIError{
				Detail: "Internal server error",
			}
			body, _ := json.Marshal(err)
			w.Write(body)
		} else if status == http.StatusTooManyRequests {
			err := &replicate.APIError{
				Detail: "Too many requests",
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

	assert.ErrorContains(t, err, "Internal server error")
}
