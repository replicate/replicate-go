package replicate

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	ErrInvalidUTF8Data = errors.New("invalid UTF-8 data")
)

const (
	// SSETypeDefault is the default type of SSEEvent.
	SSETypeDefault = "message"

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

// decodeSSEEvent parses the raw SSE event data and returns an SSEEvent pointer and an error.
func decodeSSEEvent(b string) (*SSEEvent, error) {
	chunks := []string{}
	e := &SSEEvent{Type: SSETypeDefault}

	for _, line := range strings.Split(b, "\n") {
		// Parse field and value from line
		parts := strings.SplitN(line, ":", 2)

		field := ""
		if len(parts) > 0 {
			field = parts[0]
		}

		var value string
		if len(parts) == 2 {
			value = parts[1]
			// Trim leading space if present
			value = strings.TrimLeft(value, " ")
		}

		switch field {
		case "id":
			e.ID = value
		case "event":
			e.Type = value
		case "data":
			chunks = append(chunks, value)
		default:
			// ignore
		}
	}

	e.Data = strings.Join(chunks, "\n")

	// Return nil if event data is empty and event type is not "done"
	if e.Data == "" && e.Type != SSETypeDone {
		return nil, nil
	}

	return e, nil
}

func (e *SSEEvent) String() string {
	switch e.Type {
	case SSETypeOutput:
		return e.Data
	default:
		return ""
	}
}

func (r *Client) sendError(err error, errChan chan error) {
	select {
	case errChan <- err:
	default:
	}
}

func (r *Client) Stream(ctx context.Context, identifier string, input PredictionInput, webhook *Webhook) (<-chan SSEEvent, <-chan error) {
	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	id, err := ParseIdentifier(identifier)
	if err != nil {
		r.sendError(err, errChan)
		return sseChan, errChan
	}

	var prediction *Prediction
	if id.Version == nil {
		prediction, err = r.CreatePredictionWithModel(ctx, id.Owner, id.Name, input, webhook, true)
	} else {
		prediction, err = r.CreatePrediction(ctx, *id.Version, input, webhook, true)
	}

	if err != nil {
		r.sendError(err, errChan)
		return sseChan, errChan
	}

	go r.streamPrediction(ctx, prediction, nil, sseChan, errChan)

	return sseChan, errChan
}

func (r *Client) StreamPrediction(ctx context.Context, prediction *Prediction) (<-chan SSEEvent, <-chan error) {
	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	go r.streamPrediction(ctx, prediction, nil, sseChan, errChan)

	return sseChan, errChan
}

func scanEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\n', '\n'}); i >= 0 {
		// We have a full \n\n-terminated event.
		return i + 2, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated event. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func (r *Client) streamPrediction(ctx context.Context, prediction *Prediction, lastEvent *SSEEvent, sseChan chan SSEEvent, errChan chan error) {
	url := prediction.URLs["stream"]
	if url == "" {
		r.sendError(errors.New("streaming not supported or not enabled for this prediction"), errChan)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		r.sendError(fmt.Errorf("failed to create request: %w", err), errChan)
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	if lastEvent != nil {
		req.Header.Set("Last-Event-ID", lastEvent.ID)
	}

	resp, err := r.c.Do(req)
	if err != nil {
		r.sendError(fmt.Errorf("failed to send request: %w", err), errChan)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		r.sendError(fmt.Errorf("received invalid status code: %d", resp.StatusCode), errChan)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(scanEvents)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	done := false

	for scanner.Scan() {
		event, err := decodeSSEEvent(scanner.Text())
		if err != nil {
			r.sendError(err, errChan)
			continue
		}

		if event == nil {
			// Skip empty events
			continue
		}
		lastEvent = event

		select {
		case sseChan <- *event:
		case <-ctx.Done():
			return
		}

		if event.Type == SSETypeDone {
			done = true
		}
	}

	if err := scanner.Err(); err != nil {
		if !errors.Is(err, context.Canceled) {
			r.sendError(err, errChan)
		}
	}
	if !done {
		// retry
		r.streamPrediction(ctx, prediction, lastEvent, sseChan, errChan)
		return
	}

	close(sseChan)
	close(errChan)
}
