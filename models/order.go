package models

import (
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	CustomerID uint      `json:"customerId" gorm:"column:customer_id"`
	Products   []Product `json:"products,omitempty" gorm:"-"`
}
