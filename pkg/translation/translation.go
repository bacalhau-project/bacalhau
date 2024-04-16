// Package translation provides interfaces for translating from a Job to a
// different Job. This allow us to accept more job types than we have
// executors as we translate from the abstract type to the concrete executor.
//
// When presented with a Job, this package iterates through the tasks
// belonging to the job to determine whether any of the tasks have an
// Engine type that is not one of the core executors (docker or wasm).
// If it does not, then it returns immediately.
//
// For the discovered tasks, the TranslatorProvider is asked to provide an
// implementation of the Translator interface based on the task's engine type.
// The newly obtained Translator processes the task and returns a new task
// with a known engine type (docker or wasm). Depending on where the
// translation occurs, extra work might result in the generation of a derived
// job.

package translation

import (
	"context"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/translation/translators"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// Translator defines what functions are required for a component that
// is able to translate from one job to another. It is important that
// implementers ensure that their implementation is reentrant - which
// means it should not use any mutable state after initialization.
type Translator interface {
	provider.Providable

	Translate(*models.Task) (*models.Task, error)
}

// TranslatorProvider is an alias for `provider.Provider[Translator]`
type TranslatorProvider interface {
	provider.Provider[Translator]
}

// NewStandardTranslatorsProvider returns a TranslatorProvider which maps names
// to implementations of the Translator interface
func NewStandardTranslatorsProvider() TranslatorProvider {
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

	var errs error

	for i := range newJob.Tasks {
		task := newJob.Tasks[i]
		kind := task.Engine.Type

		if models.IsDefaultEngineType(kind) {
			continue // and leave this task in place
		}

		if translator, err := provider.Get(ctx, kind); err != nil {
			errs = errors.Join(errs, err)
		} else {
			t, err := translator.Translate(task)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}

			// Copy the newly translated task over the top of the task
			// that was copied from the original job
			newJob.Tasks[i] = t
		}
	}

	return newJob, errs
}

// ShouldTranslate works out whether we need to carry on with translation, that is
// are there any engine types specified that are not a default engine and we know
// how to translate.  If not, then we can exit early.
func ShouldTranslate(ctx context.Context, provider TranslatorProvider, tasks []*models.Task) (bool, error) {
	var errs error
	needTranslationCount := 0

	for i := range tasks {
		kind := tasks[i].Engine.Type
		if provider.Has(ctx, kind) {
			needTranslationCount += 1
		} else if kind == models.EngineDocker || kind == models.EngineWasm || kind == models.EngineNoop {
			continue
		} else {
			errs = errors.Join(errs, fmt.Errorf("unknown task type identified in translation: '%s'", kind))
		}
	}

	return needTranslationCount > 0, errs
}
