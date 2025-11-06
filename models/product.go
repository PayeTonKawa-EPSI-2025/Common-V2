package models

import (
	"gorm.io/gorm"
)

type ProductDetails struct {
	Price       float32 `json:"price" gorm:"column:price"`
	Description string  `json:"description" gorm:"column:description"`
	Color       string  `json:"color" gorm:"column:color"`
}

type Product struct {
	gorm.Model
	Name    string         `json:"name" gorm:"column:name"`
	Stock   uint           `json:"stock" gorm:"column:stock"`
	Details ProductDetails `json:"details,omitempty" gorm:"embedded;embeddedPrefix:details_"`
}
