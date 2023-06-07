package objectstore

import (
	"errors"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/strings/slices"
)

var ErrUnregisteredType = func(t interface{}) error { return fmt.Errorf("%T is not registered with objectstore", t) }

type DB struct {
	location        string
	db              *gorm.DB
	registeredTypes []string
}

type DBConfig struct {
	Location string
}

func NewObjectStore(options ...Option) *DB {
	config := DBConfig{
		Location: ":memory:",
	}

	for _, opt := range options {
		opt(&config)
	}

	return &DB{
		location:        config.Location,
		registeredTypes: make([]string, 8), //nolint:gomnd
	}
}

func (d *DB) Connect(migrationTypes ...interface{}) error {
	dialector := sqlite.Open(d.location)
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}

	// Migrate the schema
	for _, m := range migrationTypes {
		err = db.AutoMigrate(m)
		if err != nil {
			return err
		}

		d.registeredTypes = append(d.registeredTypes, d.typeName(m))
	}

	d.db = db
	return nil
}

func (d *DB) Get(t interface{}, id string) bool {
	if err := d.db.First(t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
	}

	return true
}

func (d *DB) GetBy(t interface{}, field string, val interface{}) error {
	return d.db.Find(t, field, val).Error
}

func (d *DB) Create(obj interface{}) error {
	if !d.isRegistered(obj) {
		return ErrUnregisteredType(obj)
	}

	return d.db.Create(obj).Error
}

func (d *DB) Save(obj interface{}) error {
	if !d.isRegistered(obj) {
		return ErrUnregisteredType(obj)
	}

	return d.db.Save(obj).Error
}

func (d *DB) isRegistered(obj interface{}) bool {
	return slices.Contains(d.registeredTypes, d.typeName(obj))
}

func (d *DB) typeName(obj interface{}) string {
	return fmt.Sprintf("%T", obj)
}

func (d *DB) Close() {

}
