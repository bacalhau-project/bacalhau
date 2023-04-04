package persistent

import (
	/*
		use this impl instead of gorm default to negate the need of building with CGO
		we are against CGO for reasons stated in https://github.com/bacalhau-project/bacalhau/commit/07c714438ad92fc5b5b579da96ec35e94126eb6d
	*/
	"github.com/glebarez/sqlite"
	//_ "gorm.io/driver/sqlite"
	"github.com/raulk/clock"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

/*
DefaultDialect Usage details: https://www.sqlite.org/inmemorydb.html
passing in the string ":memory:" results in a new database created purely in memory.
The database ceases to exist as soon as the database connection is closed.
Every :memory: database is distinct from every other. (a requirement for current testing)
So, opening two database connections each with the filename ":memory:" will create two independent in-memory databases.

busy_timeout=5000 sets a busy handler that sleeps for a specified amount of time when a table is locked.
The handler will sleep multiple times until at least "ms" milliseconds of sleeping have accumulated.
After at least "ms" milliseconds of sleeping, the handler returns 0 which causes sqlite3_step() to return SQLITE_BUSY
We do this to prevent the error "database is locked" from being returned on parallel reads and writes.
*/
var DefaultDialect = sqlite.Open(":memory:?busy_timeout=5000")

var DefaultClock = clock.New()

var DefaultMaxOpenConns = 1
var DefaultMaxIdleConns = 1

func NewDefaultConfig() *Config {
	return &Config{
		Clock:        DefaultClock,
		Dialect:      DefaultDialect,
		Logger:       nil,
		MaxOpenConns: DefaultMaxOpenConns,
		MaxIdlConns:  DefaultMaxIdleConns,
	}
}

type Config struct {
	Clock        clock.Clock
	Dialect      gorm.Dialector
	Logger       logger.Interface
	MaxOpenConns int
	MaxIdlConns  int
}

type ConfigOpt func(cfg *Config)

func WithClock(c clock.Clock) ConfigOpt {
	return func(cfg *Config) {
		cfg.Clock = c
	}
}

func WithDialect(d gorm.Dialector) ConfigOpt {
	return func(cfg *Config) {
		cfg.Dialect = d
	}
}

func WithLogger(l logger.Interface) ConfigOpt {
	return func(cfg *Config) {
		cfg.Logger = l
	}
}

// WithMaxIdleConns sets the maximum number of connections in the idle
// connection pool.
//
// If MaxOpenConns is greater than 0 but less than the new MaxIdleConns,
// then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
//
// If n <= 0, no idle connections are retained.
//
// The default max idle connections is currently 1.
func WithMaxIdleConns(c int) ConfigOpt {
	return func(cfg *Config) {
		cfg.MaxIdlConns = c
	}
}

// WithMaxOpenConns sets the maximum number of open connections to the database.
//
// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
// MaxIdleConns, then MaxIdleConns will be reduced to match the new
// MaxOpenConns limit.
//
// If n <= 0, then there is no limit on the number of open connections.
// The default is 0 (unlimited).
func WithMaxOpenConns(c int) ConfigOpt {
	return func(cfg *Config) {
		cfg.MaxOpenConns = c
	}
}
