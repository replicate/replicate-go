package replicate

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

type WebhookSigningSecret struct {
	Key string `json:"key"`

	rawJSON json.RawMessage
}

func (wss *WebhookSigningSecret) RawJSON() json.RawMessage {
	return wss.rawJSON
}

var _ json.Unmarshaler = (*WebhookSigningSecret)(nil)

func (wss *WebhookSigningSecret) UnmarshalJSON(data []byte) error {
	wss.rawJSON = data
	type Alias WebhookSigningSecret
	alias := &struct{ *Alias }{Alias: (*Alias)(wss)}
	return json.Unmarshal(data, alias)
}

// GetDefaultWebhookSecret gets the default webhook signing secret
func (r *Client) GetDefaultWebhookSecret(ctx context.Context) (*WebhookSigningSecret, error) {
	secret := &WebhookSigningSecret{}
	err := r.fetch(ctx, http.MethodGet, "/webhooks/default/secret", nil, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get default webhook signing secret")
	}

	return secret, nil
}

// ValidateWebhookRequest validates the signature from an incoming webhook request using the provided secret
func ValidateWebhookRequest(req *http.Request, secret WebhookSigningSecret) (bool, error) {
	id := req.Header.Get("webhook-id")
	timestamp := req.Header.Get("webhook-timestamp")
	signature := req.Header.Get("webhook-signature")
	if id == "" || timestamp == "" || signature == "" {
		return false, fmt.Errorf("missing required webhook headers: id=%s, timestamp=%s, signature=%s", id, timestamp, signature)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read request body: %w", err)
	}
	defer req.Body.Close()

	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	body := string(bodyBytes)

	signedContent := fmt.Sprintf("%s.%s.%s", id, timestamp, body)

	keyParts := strings.Split(secret.Key, "_")
	if len(keyParts) != 2 {
		return false, fmt.Errorf("invalid secret key format: %s", secret.Key)
	}
	secretBytes, err := base64.StdEncoding.DecodeString(keyParts[1])
	if err != nil {
		return false, fmt.Errorf("failed to base64 decode secret key: %w", err)
	}

	h := hmac.New(sha256.New, secretBytes)
	h.Write([]byte(signedContent))
	computedSignatureBytes := h.Sum(nil)

	for _, sig := range strings.Split(signature, " ") {
		sigParts := strings.Split(sig, ",")
		if len(sigParts) < 2 {
			return false, fmt.Errorf("invalid signature format: %s", sig)
		}

		sigBytes, err := base64.StdEncoding.DecodeString(sigParts[1])
		if err != nil {
			return false, fmt.Errorf("failed to base64 decode signature: %w", err)
		}

		if hmac.Equal(sigBytes, computedSignatureBytes) {
			return true, nil
		}
	}

	return false, nil
}
