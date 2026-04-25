package contract

import "encoding/json"

// Event represents an event that can be published by the event system.
type Event struct {
	EventID string          `json:"event_id"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}
