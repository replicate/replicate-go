package sse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Backoff is a copy of replicate.Backoff to avoid import cycles
type Backoff interface {
	NextDelay(retries int) time.Duration
}

type Streamer struct {
	c          *http.Client
	url        string
	maxRetries int
	backoff    Backoff

	attempt     int
	lastEventID string

	decoder       *Decoder
	currentStream io.ReadCloser
}

func NewStreamer(c *http.Client, url string, maxRetries int, backoff Backoff) *Streamer {
	return &Streamer{
		c:          c,
		url:        url,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

var ErrMaximumRetries = errors.New("Exceeded maximum retries")

// connect (re-)establishes the connection to the SSE server. It only returns an
// error if it cannot recover through retries.
func (s *Streamer) connect(ctx context.Context) error {
	for {
		if s.attempt > s.maxRetries {
			return ErrMaximumRetries
		}

		delay := 0 * time.Second
		if s.attempt > 0 {
			// delay on connection retry
			delay = s.backoff.NextDelay(s.attempt - 1)
		}
		s.attempt++
		reconnectDelay := time.NewTimer(delay)
		// once we only support go 1.23+, we can use time.After() here and simplify
		defer reconnectDelay.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-reconnectDelay.C:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "text/event-stream")

		if s.lastEventID != "" {
			req.Header.Set("Last-Event-ID", s.lastEventID)
		}

		//nolint:bodyclose
		resp, err := s.c.Do(req)
		if err != nil {
			// try again
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received invalid status code: %d", resp.StatusCode)
		}

		if s.currentStream != nil {
			err = s.currentStream.Close()
			if err != nil {
				return err
			}
		}
		s.currentStream = resp.Body
		s.decoder = NewDecoder(s.currentStream)
		return nil
	}
}

func (s *Streamer) NextEvent(ctx context.Context) (*Event, error) {
	if s.decoder == nil {
		if err := s.connect(ctx); err != nil {
			return nil, err
		}
	}
	for {
		e, err := s.decoder.Next()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				if err = s.connect(ctx); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}
		s.lastEventID = e.ID
		return &e, nil
	}
}

func (s *Streamer) Close() error {
	if s.currentStream != nil {
		return s.currentStream.Close()
	}
	return nil
}
