package models

import "gorm.io/gorm"

type APIKey struct {
	gorm.Model
	Key    string `gorm:"unique;not null"`
	UserID uint   // Foreign key for User
	Nodes  []Node `gorm:"foreignKey:APIKeyID"`
}
