package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	// Additional fields like username, email, etc., can be added here
	APIKeys []APIKey `gorm:"foreignKey:UserID"`
}
