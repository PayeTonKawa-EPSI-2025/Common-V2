package events

import (
	"time"

	"github.com/PayeTonKawa-EPSI-2025/Common/models"
)

const (
	CustomerCreated EventType = "customer.created"
	CustomerUpdated EventType = "customer.updated"
	CustomerDeleted EventType = "customer.deleted"
)

// CustomerEvent represents the structure of a customer event
type CustomerEvent struct {
	Type      EventType       `json:"type"`
	Customer  models.Customer `json:"customer"`
	Timestamp time.Time       `json:"timestamp"`
}
