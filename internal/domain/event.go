package domain

import "time"

type Event struct {
	Type      string
	Reason    string
	Message   string
	Count     int32
	FirstSeen time.Time
	LastSeen  time.Time
	Source    string
}

func NewEvent(eventType, reason, message string) *Event {
	return &Event{
		Type:    eventType,
		Reason:  reason,
		Message: message,
	}
}

func (e *Event) IsWarning() bool {
	return e.Type == "Warning"
}

func (e *Event) IsNormal() bool {
	return e.Type == "Normal"
}
