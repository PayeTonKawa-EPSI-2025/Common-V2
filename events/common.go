package events

import (
	"encoding/json"
	"time"
)

type EventType string

type GenericEvent struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}
