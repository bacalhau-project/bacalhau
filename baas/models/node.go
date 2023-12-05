package models

import "gorm.io/gorm"

type Node struct {
	gorm.Model
	PeerID    string
	Addresses string `gorm:"type:text"` // Store as JSON string
	APIKeyID  uint   // Foreign key for APIKey
}
