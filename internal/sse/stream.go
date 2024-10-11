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

func (s *Streamer) connect(ctx context.Context) error {
	if s.attempt > s.maxRetries {
		return fmt.Errorf("Exceeded maximum retries")
	}

	delay := 0 * time.Second
	if s.attempt > 0 {
		// delay on connection retry
		delay = s.backoff.NextDelay(s.attempt - 1)
	}
	reconnectDelay := time.NewTimer(delay)
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

	resp, err := s.c.Do(req)
	if err != nil {
		return err
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
	s.attempt++
	return nil
}

func (s *Streamer) NextEvent(ctx context.Context) (*Event, error) {
	if s.decoder == nil {
		s.connect(ctx)
	}
	for {
		e, err := s.decoder.Next()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				s.connect(ctx)
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
