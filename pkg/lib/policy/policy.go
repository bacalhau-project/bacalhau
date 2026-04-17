package policy

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/v1/loader"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

// Policy is an executable bundle of Rego policy documents.
type Policy struct {
	modules []regoOpt
}

// Load a policy from the passed filesystem and path. Path can either be a
// single file, or a directory. In the latter case, all of the files in the
// directory will be loaded as policy documents and will be in scope for
// prepared queries.
func FromFS(source fs.FS, path string) (*Policy, error) {
	loaded, err := loader.NewFileLoader().WithFS(source).All([]string{path})
	if err != nil {
		return nil, err
	}

	modules := lo.Map(
		maps.Values(loaded.Modules),
		func(m *loader.RegoFile, _ int) regoOpt { return rego.ParsedModule(m.Parsed) },
	)

	return &Policy{modules: modules}, nil
}

// Load a policy from the host filesystem at path. Path can either be a single
// file, or a directory. In the latter case, all of the files in the directory
// will be loaded as policy documents and will be in scope for prepared queries.
func FromPath(path string) (*Policy, error) {
	return FromFS(os.DirFS("/"), strings.TrimLeft(path, "/."))
}

// Like FromPath, but returns a default if the path is empty.
func FromPathOrDefault(path string, def *Policy) (*Policy, error) {
	if path == "" {
		return def, nil
	} else {
		return FromPath(path)
	}
}

type regoOpt = func(*rego.Rego)

var ErrNoResult error = errors.New("the query did not return a result")

// Query is a function that will execute a policy with provided input and return
// the result.
type Query[Input, Output any] func(ctx context.Context, input Input) (Output, error)

// AddQuery prepares a query of a certain rule from the policy expecting a
// certain input type and returns a function that will execute the query when
// given input of that type.
func AddQuery[Input, Output any](runner *Policy, rule string) Query[Input, Output] {
	opts := append(runner.modules, rego.Query("data."+rule), scryptFn, rego.StrictBuiltinErrors(true))
	query := lo.Must(rego.New(opts...).PrepareForEval(context.Background()))

	return func(ctx context.Context, t Input) (Output, error) {
		var out Output

		tracer := topdown.NewBufferTracer()
		defer func() {
			// Output tracing information, but only if the log level is appropriate
			// So we avoid going into a long loop of no-ops
			const logAt zerolog.Level = zerolog.TraceLevel
			if logger := log.Ctx(ctx); logger.GetLevel() <= logAt {
				buf := strings.Builder{}
				topdown.PrettyTraceWithLocation(&buf, *tracer)

				for _, event := range strings.Split(buf.String(), "\n") {
					logger.WithLevel(logAt).Msg(event)
				}
			}
		}()

		result, err := query.Eval(ctx, rego.EvalInput(t), rego.EvalQueryTracer(tracer))
		if err != nil {
			return out, err
		}

		if len(result) < 1 {
			return out, ErrNoResult
		}

		out = (result[0].Expressions[0].Value).(Output)
		return out, nil
	}
}
