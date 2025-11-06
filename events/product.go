package events

import (
	"time"

	"github.com/PayeTonKawa-EPSI-2025/Common/models"
)

const (
	ProductCreated EventType = "product.created"
	ProductUpdated EventType = "product.updated"
	ProductDeleted EventType = "product.deleted"
)

type ProductEvent struct {
	Type      EventType      `json:"type"`
	Product   models.Product `json:"product"`
	Timestamp time.Time      `json:"timestamp"`
}
