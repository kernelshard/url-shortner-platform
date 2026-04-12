package event

import "encoding/json"

// Event represents an event that can be published by the event system.
type Event struct {
	ID   string          `json:"event_id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
