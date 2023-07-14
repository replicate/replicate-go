package replicate

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
