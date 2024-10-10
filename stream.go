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

	"github.com/launchdarkly/eventsource"
	"github.com/replicate/replicate-go/streaming"
	"github.com/vincent-petithory/dataurl"
	"golang.org/x/sync/errgroup"
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
func decodeSSEEvent(b []byte) (*SSEEvent, error) {
	chunks := [][]byte{}
	e := &SSEEvent{Type: SSETypeDefault}

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
			value = bytes.TrimPrefix(value, []byte(" "))
		}

		switch field {
		case "id":
			e.ID = string(value)
		case "event":
			e.Type = string(value)
		case "data":
			chunks = append(chunks, value)
		default:
			// ignore
		}
	}

	data := bytes.Join(chunks, []byte("\n"))
	if !utf8.Valid(data) {
		return nil, ErrInvalidUTF8Data
	}
	e.Data = string(data)

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

	r.streamPrediction(ctx, prediction, nil, sseChan, errChan)

	return sseChan, errChan
}

func (r *Client) StreamPrediction(ctx context.Context, prediction *Prediction) (<-chan SSEEvent, <-chan error) {
	sseChan := make(chan SSEEvent, 64)
	errChan := make(chan error, 64)

	r.streamPrediction(ctx, prediction, nil, sseChan, errChan)

	return sseChan, errChan
}

func (r *Client) StreamPredictionFiles(ctx context.Context, prediction *Prediction) (<-chan streaming.File, error) {
	url := prediction.URLs["stream"]
	if url == "" {
		return nil, errors.New("streaming not supported or not enabled for this prediction")
	}

	ch := make(chan streaming.File)

	go r.streamFilesTo(ctx, ch, url, "")
	return ch, nil
}

func (r *Client) StreamPredictionText(ctx context.Context, prediction *Prediction) (io.Reader, error) {
	url := prediction.URLs["stream"]
	if url == "" {
		return nil, errors.New("streaming not supported or not enabled for this prediction")
	}

	reader, writer := io.Pipe()

	go r.streamTextTo(ctx, writer, url, "")
	return reader, nil
}

func (r *Client) streamTextTo(ctx context.Context, writer *io.PipeWriter, url string, lastEventID string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		writer.CloseWithError(fmt.Errorf("failed to create request: %w", err))
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	resp, err := r.c.Do(req)
	if err != nil {
		writer.CloseWithError(fmt.Errorf("failed to send request: %w", err))
		return
	}

	if resp.StatusCode != http.StatusOK {
		writer.CloseWithError(fmt.Errorf("received invalid status code: %d", resp.StatusCode))
		return
	}
	defer resp.Body.Close()
	decoder := eventsource.NewDecoder(resp.Body)
	for {
		event, err := decoder.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				// retry (TODO: backoff policy?)
				r.streamTextTo(ctx, writer, url, lastEventID)
				return
			}
			writer.CloseWithError(fmt.Errorf("Failed to get token: %w", err))
			return
		}
		lastEventID = event.Id()
		switch event.Event() {
		case SSETypeOutput:
			io.WriteString(writer, event.Data())
		case SSETypeDone:
			writer.Close()
			return
		case SSETypeLogs:
			// TODO
		default:
			writer.CloseWithError(fmt.Errorf("unknown event type %s", event.Event()))
			return
		}
	}
}

type dataURL struct {
	url string
}

var _ streaming.File = &dataURL{}

func (d *dataURL) Body(_ context.Context) (io.ReadCloser, error) {
	data, err := dataurl.DecodeString(d.url)

	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(data.Data)), nil
}

type httpURL struct {
	c   *http.Client
	url string
}

var _ streaming.File = &httpURL{}

func (h *httpURL) Body(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

type errWrapper struct {
	err error
}

var _ streaming.File = &errWrapper{}

func fileError(err error) streaming.File {
	return &errWrapper{err: err}
}

func (e *errWrapper) Body(_ context.Context) (io.ReadCloser, error) {
	return nil, e.err
}

func (e *errWrapper) Close() error {
	return nil
}

func (r *Client) streamFilesTo(ctx context.Context, out chan<- streaming.File, url string, lastEventID string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		out <- fileError(err)
		close(out)
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	resp, err := r.c.Do(req)
	if err != nil {
		out <- fileError(fmt.Errorf("failed to send request: %w", err))
		close(out)
		return
	}

	if resp.StatusCode != http.StatusOK {
		out <- fileError(fmt.Errorf("received invalid status code: %d", resp.StatusCode))
		close(out)
		return
	}
	defer resp.Body.Close()
	decoder := eventsource.NewDecoder(resp.Body)
	for {
		event, err := decoder.Decode()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				// retry (TODO: backoff policy?)
				r.streamFilesTo(ctx, out, url, lastEventID)
				return
			}
			out <- fileError(fmt.Errorf("Failed to get token: %w", err))
			close(out)
			return
		}
		lastEventID = event.Id()
		switch event.Event() {
		case SSETypeOutput:
			if strings.HasPrefix(event.Data(), "data:") {
				out <- &dataURL{url: event.Data()}
			} else if strings.HasPrefix(event.Data(), "http") {
				out <- &httpURL{c: r.c, url: event.Data()}
			}
		case SSETypeDone:
			close(out)
			return
		case SSETypeLogs:
			// TODO
		default:
			out <- fileError(fmt.Errorf("unknown event type %s", event.Event()))
			return
		}
	}

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

	if resp.StatusCode != http.StatusOK {
		r.sendError(fmt.Errorf("received invalid status code: %d", resp.StatusCode), errChan)
		return
	}

	reader := bufio.NewReader(resp.Body)
	var buf bytes.Buffer
	lineChan := make(chan []byte)

	g, ctx := errgroup.WithContext(ctx)
	done := make(chan struct{})

	g.Go(func() error {
		defer close(lineChan)
		defer resp.Body.Close()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-done:
				return nil
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					return err
				}
				select {
				case lineChan <- line:
				case <-ctx.Done():
					return ctx.Err()
				}
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

					event, err := decodeSSEEvent(b)
					if err != nil {
						r.sendError(err, errChan)
						continue
					}

					if event == nil {
						// Skip empty events
						continue
					}

					select {
					case sseChan <- *event:
					case <-done:
						return
					case <-ctx.Done():
						return
					}

					if event.Type == SSETypeDone {
						close(done)
						return
					}
				}
			}
		}
	}()

	go func() {
		err := g.Wait()

		if err != nil {
			if errors.Is(err, io.EOF) {
				select {
				case <-done:
					// if we get EOF after receiving "done", we're done
					return
				default:
				}
				// Attempt to reconnect if the connection was closed before the stream was done
				r.streamPrediction(ctx, prediction, lastEvent, sseChan, errChan)
				return
			}

			if !errors.Is(err, context.Canceled) {
				r.sendError(err, errChan)
			}
		}

		close(sseChan)
		close(errChan)
	}()
}
