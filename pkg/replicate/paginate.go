package replicate

import (
	"context"
)

// Page represents a paginated response from Replicate's API.
type Page[T any] struct {
	Previous *string `json:"previous,omitempty"`
	Next     *string `json:"next,omitempty"`
	Results  []T     `json:"results"`
}

// Paginate takes a Page and the Client request method, and iterates through pages of results.
func Paginate[T any](ctx context.Context, client *Client, initialPage *Page[T]) (<-chan []T, <-chan error) {
	resultsChan := make(chan []T)
	errChan := make(chan error)

	go func() {
		defer close(resultsChan)
		defer close(errChan)

		resultsChan <- initialPage.Results
		nextURL := initialPage.Next

		for nextURL != nil {
			page := &Page[T]{}
			err := client.request(ctx, "GET", *nextURL, nil, page)
			if err != nil {
				errChan <- err
				return
			}

			resultsChan <- page.Results

			nextURL = page.Next
		}
	}()

	return resultsChan, errChan
}
