package models

import (
	"gorm.io/gorm"
)

type Address struct {
	PostalCode string `json:"postalCode" gorm:"column:postal_code"`
	City       string `json:"city" gorm:"column:city"`
}

type Profile struct {
	FirstName string `json:"firstName" gorm:"column:first_name"`
	LastName  string `json:"lastName" gorm:"column:last_name"`
}

type Company struct {
	CompanyName string `json:"companyName" gorm:"column:company_name"`
}

type Customer struct {
	gorm.Model
	Username  string `json:"username" gorm:"column:username"`
	FirstName string `json:"firstName" gorm:"column:first_name"`
	LastName  string `json:"lastName" gorm:"column:last_name"`
	Name      string `json:"name" gorm:"column:name"`

	Address Address `json:"address,omitempty" gorm:"embedded;embeddedPrefix:address_"`
	Profile Profile `json:"profile,omitempty" gorm:"embedded;embeddedPrefix:profile_"`
	Company Company `json:"company,omitempty" gorm:"embedded;embeddedPrefix:company_"`

	Orders []Order `json:"orders,omitempty" gorm:"-"`
}
