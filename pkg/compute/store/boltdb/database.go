package boltdb

import (
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	DefaultDatabasePermissions = 0600
)

func GetDatabase(path string) (*bolt.DB, error) {
	database, err := bolt.Open(path, DefaultDatabasePermissions, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, err
	}
	return database, nil
}
