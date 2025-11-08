package events

import (
	"time"
)

const (
	OrderCreated EventType = "order.created"
	OrderUpdated EventType = "order.updated"
	OrderDeleted EventType = "order.deleted"
)

type OrderEvent struct {
	Type      EventType       `json:"type"`
	Order     SimplifiedOrder `json:"order"`
	Timestamp time.Time       `json:"timestamp"`
}

type SimplifiedOrder struct {
	OrderID    uint   `json:"orderId"`
	CustomerID uint   `json:"customerId"`
	ProductIDs []uint `json:"productIds,omitempty"`
}
