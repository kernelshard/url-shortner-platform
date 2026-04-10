package event

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const maxRetries = 3
const baseDelay = 100 * time.Millisecond

type HTTPPublisher struct{}

func NewHTTPPublisher() *HTTPPublisher {
	return &HTTPPublisher{}
}

// Publish publishes the given event via HTTP to the notification service.
func (p *HTTPPublisher) Publish(ctx context.Context, event Event) error {
	log.Printf("publishing event: type=%s", event.Type)
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	for i := range maxRetries {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			"http://notification-service:8081/events",
			bytes.NewBuffer(body),
		)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}

		if err != nil {
			log.Printf("event publish failed (attempt=%d, type=%s): %v", i+1, event.Type, err)
		} else {
			log.Printf("event publish failed (attempt=%d, type=%s): status=%d", i+1, event.Type, resp.StatusCode)
			resp.Body.Close()
		}

		time.Sleep(baseDelay * time.Duration(1<<i))
	}

	log.Printf("event permanently failed (type=%s) after %d attempts", event.Type, maxRetries)
	return nil
}
