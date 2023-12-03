package replicate

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/sync/errgroup"
)

var (
	ErrInvalidUTF8Data = errors.New("invalid UTF-8 data")
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Type string
	ID   string
	Data string
}

func (e *SSEEvent) decode(b []byte, sb *strings.Builder) error {
	for _, line := range bytes.Split(b, []byte("\n")) {
		// Parse field and value from line
		parts := bytes.SplitN(line, []byte{':'}, 2)
		field := string(parts[0])
		var value []byte
		if len(parts) == 2 {
			value = parts[1]
			// Trim leading space if present
			value, _ = bytes.CutPrefix(value, []byte(" "))
		}

		switch field {
		case "id":
			e.ID = string(value)
		case "event":
			e.Type = string(value)
		case "data":
			if sb.Len() > 0 {
				sb.WriteRune('\n')
			}
			sb.Write(value)
		default:
			// ignore
		}
	}

	data := sb.String()
	sb.Reset()

	if !utf8.ValidString(data) {
		return ErrInvalidUTF8Data
	}

	e.Data = data

	return nil
}

// Stream runs a model with the given input and returns a streams its output.
func (r *Client) Stream(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (<-chan SSEEvent, <-chan error, error) {
	id, err := ParseIdentifier(identifier)
	if err != nil {
		return nil, nil, err
	}

	if id.Version == nil {
		return nil, nil, errors.New("version must be specified")
	}

	prediction, err := r.CreatePrediction(ctx, *id.Version, input, webhook, true)
	if err != nil {
		return nil, nil, err
	}

	url := prediction.URLs["stream"]
	if url == "" {
		return nil, nil, errors.New("streaming not supported")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer close(sseChan)

		resp, err := r.c.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received invalid status code %d", resp.StatusCode)
		}

		r := bufio.NewReader(resp.Body)
		var buf bytes.Buffer
		lineChan := make(chan []byte)

		g.Go(func() error {
			for {
				line, err := r.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				lineChan <- line
			}
		})

		sb := strings.Builder{}
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case b := <-lineChan:
				buf.Write(b)

				if bytes.Equal(b, []byte("\n")) {
					b := buf.Bytes()
					buf.Reset()

					event := SSEEvent{Type: "message"}
					if err := event.decode(b, &sb); err != nil {
						errChan <- err
					}

					switch event.Type {
					case "error":
						errChan <- unmarshalAPIError([]byte(event.Data))
					case "done":
						return nil
					default:
						sseChan <- event
					}
				}
			}
		}
	})

	go func() {
		err := g.Wait()
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	return sseChan, errChan, nil
}
