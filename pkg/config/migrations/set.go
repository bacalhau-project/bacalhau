package migrations

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var set migrationList

func GetMigrations() ([]Migration, error) {
	return set.GetMigrations()
}

//nolint:gochecknoinits
func init() {
	set = NewMigrationList()
}

type Migration interface {
	Sequence() int
	Migrate(config types.BacalhauConfig) (types.BacalhauConfig, error)
}

type mig struct {
	seq       int
	migration MigrationFn
}

func (m mig) Sequence() int {
	return m.seq
}

func (m mig) Migrate(config types.BacalhauConfig) (types.BacalhauConfig, error) {
	return m.migration(config)
}

type migrationList struct {
	ms map[int]mig
}

func NewMigrationList() migrationList {
	return migrationList{map[int]mig{}}
}

// Register adds a migration to the migration list. This should be called in an init function.
func (ml *migrationList) Register(seq int, mf MigrationFn) {
	if seq <= 0 {
		panic(fmt.Sprintf("invalid migration number: %d", seq))
	}

	if _, exists := ml.ms[seq]; exists {
		panic(fmt.Sprintf("duplicate migration registered: %d", seq))
	}

	ml.ms[seq] = mig{
		seq:       seq,
		migration: mf,
	}
}

type MigrationFn = func(config types.BacalhauConfig) (types.BacalhauConfig, error)

func (ml *migrationList) GetMigrations() ([]Migration, error) {
	// Check patch list is consistent with no gaps
	count := len(ml.ms)

	// migration 0 must not exist - it's the base schema by definition
	if _, exists := ml.ms[0]; exists {
		return nil, fmt.Errorf("found migration 0, which should not exist")
	}

	// index from 1 since schema seq 0 is the base and not in `ml`
	for i := 1; i <= count; i++ {
		if _, exists := ml.ms[i]; !exists {
			return nil, fmt.Errorf("missing migration %d", i)
		}
	}

	migs := make([]Migration, 0, count)
	for i := 1; i <= count; i++ {
		migs = append(migs, ml.ms[i])
	}
	return migs, nil
}
