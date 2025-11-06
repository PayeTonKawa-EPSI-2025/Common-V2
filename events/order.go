package events

import (
	"time"

	"github.com/PayeTonKawa-EPSI-2025/Common/models"
)

const (
	OrderCreated EventType = "order.created"
	OrderUpdated EventType = "order.updated"
	OrderDeleted EventType = "order.deleted"
)

type OrderEvent struct {
	Type      EventType    `json:"type"`
	Order     models.Order `json:"order"`
	Timestamp time.Time    `json:"timestamp"`
}
