package replicate_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/replicate/replicate-go"
)

func TestStreamText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `event: output
data: foo

event: done

`)
	}))
	t.Cleanup(ts.Close)

	p := &replicate.Prediction{
		URLs: map[string]string{
			"stream": ts.URL,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	c, err := replicate.NewClient(replicate.WithToken("test-token"))
	require.NoError(t, err)

	r, err := c.StreamPredictionText(ctx, p)
	t.Cleanup(func() { r.Close() })

	require.NoError(t, err)

	text, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "foo", string(text))
}

func TestStreamTextWithComment(t *testing.T) {
	// nchan seems to put a `: hi` empty event at the start of each stream.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `: hi

event: output
data: foo

event: done

`)
	}))
	t.Cleanup(ts.Close)

	p := &replicate.Prediction{
		URLs: map[string]string{
			"stream": ts.URL,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	c, err := replicate.NewClient(replicate.WithToken("test-token"))
	require.NoError(t, err)

	r, err := c.StreamPredictionText(ctx, p)
	t.Cleanup(func() { r.Close() })

	require.NoError(t, err)

	text, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "foo", string(text))
}

func TestStreamTextWithRetries(t *testing.T) {
	request := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if request == 0 {
			// first request: we return the first event
			fmt.Fprint(w, `event: output
data: foo
id: 1

`)
			request++
			return
		}

		// subsequent requests: we return the full stream, respecting Last-Event-ID
		if r.Header.Get("Last-Event-ID") != "1" {
			fmt.Fprint(w, `event: output
data: foo
id: 1

`)

		}
		fmt.Fprint(w, `event: output
data: bar
id: 2

event: done
id: 3

`)
	}))
	t.Cleanup(ts.Close)

	p := &replicate.Prediction{
		URLs: map[string]string{
			"stream": ts.URL,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	c, err := replicate.NewClient(replicate.WithToken("test-token"))
	require.NoError(t, err)

	r, err := c.StreamPredictionText(ctx, p)
	require.NoError(t, err)
	t.Cleanup(func() { r.Close() })

	text, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "foobar", string(text))
}

func TestStreamFiles(t *testing.T) {
	var baseURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/file" {
			fmt.Fprintln(w, "mango")
			return
		}
		fmt.Fprint(w, `event: output
data: data:text/plain,banana

event: output
data: data:text/plain;base64,YXBwbGU=

event: output
data: `+baseURL+`/file

event: done

`)
	}))
	t.Cleanup(ts.Close)
	baseURL = ts.URL

	p := &replicate.Prediction{
		URLs: map[string]string{
			"stream": ts.URL,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	c, err := replicate.NewClient(replicate.WithToken("test-token"))
	require.NoError(t, err)

	files, err := c.StreamPredictionFiles(ctx, p)

	require.NoError(t, err)

	// first file is a data URI
	file, err := files.NextFile(ctx)
	require.NoError(t, err)
	body, err := file.Body(ctx)
	require.NoError(t, err)
	content1, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "banana", string(content1))

	// second file is a base64'd data URI
	file, err = files.NextFile(ctx)
	require.NoError(t, err)
	body, err = file.Body(ctx)
	require.NoError(t, err)
	content2, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "apple", string(content2))

	// third file is an http URI
	file, err = files.NextFile(ctx)
	require.NoError(t, err)
	body, err = file.Body(ctx)
	require.NoError(t, err)
	content3, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "mango\n", string(content3))
}
