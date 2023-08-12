package replicate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Webhook struct {
	URL    string
	Events []WebhookEventType
}

type WebhookEventType string

const (
	WebhookEventStart     WebhookEventType = "start"
	WebhookEventOutput    WebhookEventType = "output"
	WebhookEventLogs      WebhookEventType = "logs"
	WebhookEventCompleted WebhookEventType = "completed"
)

var WebhookEventAll = []WebhookEventType{
	WebhookEventStart,
	WebhookEventOutput,
	WebhookEventLogs,
	WebhookEventCompleted,
}

func (w WebhookEventType) String() string {
	return string(w)
}

// Download
// output  is urls of Prediction.Output
// dirPath is directory for saving images
func Download(ctx context.Context, output []string, dirPath string) error {
	errChan := make(chan error)
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}

	go func() {
		var wg sync.WaitGroup
		defer close(errChan)
		for _, o := range output {
			wg.Add(1)
			o := o
			go func() {
				defer wg.Done()
				res, err := http.Get(o)
				if err != nil {
					errChan <- err
					return
				}

				defer res.Body.Close()

				// if err is nil, return error
				if _, err := os.Stat(dirPath + "/" + filepath.Base(o)); err == nil {
					errChan <- fmt.Errorf("%s image file already exists", filepath.Base(o))
					return
				}

				out, err := os.Create(dirPath + "/" + filepath.Base(o))
				if err != nil {
					errChan <- err
					return
				}
				defer out.Close()

				if _, err := io.Copy(out, res.Body); err != nil {
					errChan <- err
					return
				}
			}()
		}
		wg.Wait()
	}()

	for {
		select {
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
