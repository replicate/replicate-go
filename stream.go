package replicate

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func (e *SSEEvent) decode(b []byte) error {
	data := [][]byte{}
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
			data = append(data, value)
		default:
			// ignore
		}
	}

	if !utf8.Valid(bytes.Join(data, []byte("\n"))) {
		return ErrInvalidUTF8Data
	}

	e.Data = string(bytes.Join(data, []byte("\n")))

	return nil
}

// Stream runs a model with the given input and returns a streams its output.
func (r *Client) Stream(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (<-chan SSEEvent, <-chan error) {
	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	done := make(chan struct{})

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		id, err := ParseIdentifier(identifier)
		if err != nil {
			return err
		}

		var prediction *Prediction
		if id.Version == nil {
			prediction, err = r.CreatePredictionWithModel(ctx, id.Owner, id.Name, input, webhook, true)
		} else {
			prediction, err = r.CreatePrediction(ctx, *id.Version, input, webhook, true)
		}

		if err != nil {
			return err
		}

		url := prediction.URLs["stream"]
		if url == "" {
			return errors.New("streaming not supported")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		resp, err := r.c.Do(req)
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received invalid status code %d", resp.StatusCode)
		}

		r := bufio.NewReader(resp.Body)
		var buf bytes.Buffer
		lineChan := make(chan []byte)

		g.Go(func() error {
			defer close(lineChan)

			for {
				select {
				case <-done:
					return nil
				default:
					line, err := r.ReadBytes('\n')
					if err != nil {
						defer resp.Body.Close()
						if err == io.EOF {
							return nil
						}
						return err
					}
					lineChan <- line
				}
			}
		})

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case b, ok := <-lineChan:
				if !ok {
					return nil
				}

				buf.Write(b)

				if bytes.Equal(b, []byte("\n")) {
					b := buf.Bytes()
					buf.Reset()

					event := SSEEvent{Type: "message"}
					if err := event.decode(b); err != nil {
						errChan <- err
					}

					switch event.Type {
					case "error":
						errChan <- unmarshalAPIError([]byte(event.Data))
					case "done":
						close(done)
						return nil
					default:
						sseChan <- event
					}
				}
			}
		}
	})

	go func() {
		defer close(sseChan)
		defer close(errChan)

		err := g.Wait()
		if err != nil {
			errChan <- err
		}
	}()

	return sseChan, errChan
}
