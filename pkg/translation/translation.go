// Package translation provides interfaces for translating from a Job to a
// different Job.  This is triggered by the presence of an Engine type that
// is not one of the core executors (docker or wasm).
package translation

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/translation/translators"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/hashicorp/go-multierror"
)

// Translator defines what functions are required for a component that
// is able to translate from one job to another. It is important that
// implementors ensure that their implementation is reentrant - which
// means it should not use any mutable state after initialisation.
type Translator interface {
	provider.Providable

	Translate(*models.Task) (*models.Task, error)
}

// TranslatorProvider is an alias for `provider.Provider[Translator]`
type TranslatorProvider interface {
	provider.Provider[Translator]
}

// NewStandardTranslators returns a TranslatorProvider which maps names
// to implementations of the Translator interface
func NewStandardTranslators() TranslatorProvider {
	return provider.NewMappedProvider(map[string]Translator{
		"python": &translators.PythonTranslator{},
		"duckdb": &translators.DuckDBTranslator{},
	})
}

// Translate attempts to translate from one job to another, based on the engine type
// of the tasks in the job. After ensuring that each of the tasks is either a default
// (docker, wasm) or available via the provider, then a new Job is cloned from the
// original and the individual tasks updated.
func Translate(ctx context.Context, provider TranslatorProvider, original *models.Job) (*models.Job, error) {
	if shouldTr, err := ShouldTranslate(ctx, provider, original.Tasks); err != nil {
		return nil, err
	} else {
		// Nothing for us to do so we should return immediately
		if !shouldTr {
			return nil, nil
		}
	}

	newJob := original.Copy()
	newJob.ID = idgen.NewJobID()

	errs := new(multierror.Error)

	for i := range newJob.Tasks {
		task := newJob.Tasks[i]
		kind := task.Engine.Type

		if models.IsDefaultEngineType(kind) {
			continue // and leave this task in place
		}

		if translator, err := provider.Get(ctx, kind); err != nil {
			errs = multierror.Append(errs, err)
		} else {
			t, err := translator.Translate(task)
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}

			// Copy the newly translated task over the top of the task
			// that was copied from the original job
			newJob.Tasks[i] = t
		}
	}

	return newJob, errs.ErrorOrNil()
}

// ShouldTranslate works out whether we need to carry on with translation, that is
// are there any engine types specified that are not a default engine and we know
// how to translate.  If not, then we can exit early.
func ShouldTranslate(ctx context.Context, provider TranslatorProvider, tasks []*models.Task) (bool, error) {
	errs := new(multierror.Error)
	needTranslationCount := 0

	for i := range tasks {
		kind := tasks[i].Engine.Type
		if provider.Has(ctx, kind) {
			needTranslationCount += 1
		} else if kind == models.EngineDocker || kind == models.EngineWasm || kind == models.EngineNoop {
			continue
		} else {
			errs = multierror.Append(errs, fmt.Errorf("unknown task type identified in translation: '%s'", kind))
		}
	}

	return needTranslationCount > 0, errs.ErrorOrNil()
}
