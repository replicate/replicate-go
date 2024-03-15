package replicate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Account struct {
	Type      string `json:"type"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	GithubURL string `json:"github_url"`

	rawJSON json.RawMessage `json:"-"`
}

func (a *Account) RawJSON() json.RawMessage {
	return a.rawJSON
}

var _ json.Unmarshaler = (*Account)(nil)

func (a *Account) UnmarshalJSON(data []byte) error {
	a.rawJSON = data
	type Alias Account
	alias := &struct{ *Alias }{Alias: (*Alias)(a)}
	return json.Unmarshal(data, alias)
}

// GetCurrentAccount returns the authenticated user or organization.
func (r *Client) GetCurrentAccount(ctx context.Context) (*Account, error) {
	response := &Account{}
	err := r.fetch(ctx, http.MethodGet, "/account", nil, response)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return response, nil
}
