package event

// Event represents an event that can be published by the event system.
type Event struct {
	Type string
	Data any
}
