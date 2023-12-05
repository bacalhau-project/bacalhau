package store

import (
	"github.com/raulk/clock"
	"gorm.io/gorm"

	"github.com/bacalhau-project/bacalhau/experimental/baas/models"
)

type Store struct {
	DB    *gorm.DB
	clock clock.Clock
}

func New(opts ...ConfigOpt) (*Store, error) {
	cfg := NewDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	db, err := gorm.Open(cfg.Dialect, &gorm.Config{
		// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
		// You can disable it by setting `SkipDefaultTransaction` to true
		SkipDefaultTransaction: true,
		Logger:                 cfg.Logger,
		NowFunc:                cfg.Clock.Now,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdlConns)

	// init and migrate models
	if err := db.AutoMigrate(&models.APIKey{}, &models.User{}, &models.Node{}); err != nil {
		return nil, err
	}
	return &Store{DB: db, clock: cfg.Clock}, nil
}
