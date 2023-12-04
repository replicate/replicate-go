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

const (
	// SSETypeDone is the type of SSEEvent that indicates the prediction is done. The Data field will contain an empty JSON object.
	SSETypeDone = "done"

	// SSETypeError is the type of SSEEvent that indicates an error occurred during the prediction. The Data field will contain JSON with the error.
	SSETypeError = "error"

	// SSETypeLogs is the type of SSEEvent that contains logs from the prediction.
	SSETypeLogs = "logs"

	// SSETypeOutput is the type of SSEEvent that contains output from the prediction.
	SSETypeOutput = "output"
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

		field := ""
		if len(parts) > 0 {
			field = string(parts[0])
		}

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

func (e *SSEEvent) String() string {
	switch e.Type {
	case "output":
		return e.Data
	default:
		return ""
	}
}

func (r *Client) Stream(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (<-chan SSEEvent, <-chan error) {
	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	id, err := ParseIdentifier(identifier)
	if err != nil {
		errChan <- err
		return sseChan, errChan
	}

	var prediction *Prediction
	if id.Version == nil {
		prediction, err = r.CreatePredictionWithModel(ctx, id.Owner, id.Name, input, webhook, true)
	} else {
		prediction, err = r.CreatePrediction(ctx, *id.Version, input, webhook, true)
	}

	if err != nil {
		errChan <- err
		return sseChan, errChan
	}

	r.streamPrediction(ctx, prediction, nil, sseChan, errChan)

	return sseChan, errChan
}

func (r *Client) StreamPrediction(ctx context.Context, prediction *Prediction) (<-chan SSEEvent, <-chan error) {
	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	r.streamPrediction(ctx, prediction, nil, sseChan, errChan)

	return sseChan, errChan
}

func (r *Client) streamPrediction(ctx context.Context, prediction *Prediction, lastEvent *SSEEvent, sseChan chan SSEEvent, errChan chan error) {
	g, ctx := errgroup.WithContext(ctx)

	url := prediction.URLs["stream"]
	if url == "" {
		errChan <- errors.New("streaming not supported or not enabled for this prediction")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		errChan <- fmt.Errorf("failed to create request: %w", err)
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	if lastEvent != nil {
		req.Header.Set("Last-Event-ID", lastEvent.ID)
	}

	resp, err := r.c.Do(req)
	if err != nil || resp == nil {
		if resp != nil {
			resp.Body.Close()
		}
		errChan <- fmt.Errorf("failed to send request: %w", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Errorf("received invalid status code: %d", resp.StatusCode)
		return
	}

	done := make(chan struct{})

	reader := bufio.NewReader(resp.Body)
	var buf bytes.Buffer
	lineChan := make(chan []byte)

	g.Go(func() error {
		defer close(lineChan)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-done:
				return nil
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					defer resp.Body.Close()
					return err
				}
				lineChan <- line
			}
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case b, ok := <-lineChan:
				if !ok {
					return
				}

				buf.Write(b)

				if bytes.Equal(b, []byte("\n")) {
					b := buf.Bytes()
					buf.Reset()

					event := SSEEvent{}
					if err := event.decode(b); err != nil {
						errChan <- err
					}

					sseChan <- event
					if event.Type == SSETypeDone {
						close(done)
						return
					}
				}
			}
		}
	}()

	go func() {
		defer close(sseChan)
		defer close(errChan)

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			default:
				err := g.Wait()
				if err != nil {
					if err == io.EOF {
						// Attempt to reconnect if the connection was closed before the stream was done
						r.streamPrediction(ctx, prediction, lastEvent, sseChan, errChan)
						continue
					}

					if errors.Is(err, context.Canceled) {
						return
					}

					errChan <- err
				}
			}
		}
	}()
}
