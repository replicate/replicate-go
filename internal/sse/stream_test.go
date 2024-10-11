package sse_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/replicate/replicate-go"
	"github.com/replicate/replicate-go/internal/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `event: output
data: foo

event: done

`)
	}))
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	s := sse.NewStreamer(http.DefaultClient, ts.URL, 0, &replicate.ConstantBackoff{})
	t.Cleanup(func() { s.Close() })

	e, err := s.NextEvent(ctx)

	require.NoError(t, err)
	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "foo\n", e.Data)

	e, err = s.NextEvent(ctx)

	require.NoError(t, err)
	assert.Equal(t, "done", e.Type)
	assert.Equal(t, "", e.Data)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	s := sse.NewStreamer(http.DefaultClient, ts.URL, 0, &replicate.ConstantBackoff{})
	t.Cleanup(func() { s.Close() })

	e, err := s.NextEvent(ctx)

	// the comment event becomes a zero object
	require.NoError(t, err)
	assert.Equal(t, "", e.Type)
	assert.Equal(t, "", e.Data)

	e, err = s.NextEvent(ctx)

	require.NoError(t, err)
	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "foo\n", e.Data)

	e, err = s.NextEvent(ctx)

	require.NoError(t, err)
	assert.Equal(t, "done", e.Type)
	assert.Equal(t, "", e.Data)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	s := sse.NewStreamer(http.DefaultClient, ts.URL, 1, &replicate.ConstantBackoff{})
	t.Cleanup(func() { s.Close() })

	e, err := s.NextEvent(ctx)

	// the comment event becomes a zero object
	require.NoError(t, err)
	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "foo\n", e.Data)
	assert.Equal(t, "1", e.ID)

	e, err = s.NextEvent(ctx)

	require.NoError(t, err)
	assert.Equal(t, "output", e.Type)
	assert.Equal(t, "bar\n", e.Data)
	assert.Equal(t, "2", e.ID)

	e, err = s.NextEvent(ctx)

	require.NoError(t, err)
	assert.Equal(t, "done", e.Type)
	assert.Equal(t, "", e.Data)
	assert.Equal(t, "3", e.ID)
}
