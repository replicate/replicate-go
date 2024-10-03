package replicate_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/replicate/replicate-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `event: output
data: foo

event: done

`)
	}))
	defer ts.Close()

	p := &replicate.Prediction{
		URLs: map[string]string{
			"stream": ts.URL,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := replicate.NewClient(replicate.WithToken("test-token"))
	require.NoError(t, err)

	r, err := c.StreamPredictionText(ctx, p)

	assert.NoError(t, err)

	text, err := io.ReadAll(r)
	assert.NoError(t, err)
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
	defer ts.Close()

	p := &replicate.Prediction{
		URLs: map[string]string{
			"stream": ts.URL,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, err := replicate.NewClient(replicate.WithToken("test-token"))
	require.NoError(t, err)

	r, err := c.StreamPredictionText(ctx, p)

	assert.NoError(t, err)

	text, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "foobar", string(text))
}
