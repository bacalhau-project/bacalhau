package nodefx

import (
	"reflect"

	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/compute"
	"github.com/bacalhau-project/bacalhau/pkg/nodefx/requester"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type Settings struct {
	isCompute   bool
	isRequester bool
	repo        *repo.FsRepo
	config      *config.Config

	options map[interface{}]fx.Option
}

type Option func(s *Settings) error

// Options groups multiple options into one
func Options(opts ...Option) Option {
	return func(s *Settings) error {
		for _, opt := range opts {
			if err := opt(s); err != nil {
				return err
			}
		}
		return nil
	}
}

// Override option changes constructor for a given type
func Override(typ, constructor interface{}) Option {
	return func(s *Settings) error {
		// As is an Annotation that annotates the result of a function (i.e. a
		// constructor) to be provided as another interface.
		//
		// For example, the following code specifies that the return type of
		// bytes.NewBuffer (bytes.Buffer) should be provided as io.Writer type:
		//
		//	fx.Provide(
		//	  fx.Annotate(bytes.NewBuffer(...), fx.As(new(io.Writer)))
		//	)

		rt := reflect.TypeOf(typ).Elem()
		s.options[rt] = fx.Provide(constructor, fx.As(typ))
		return nil
	}
}

func Config(c *config.Config) Option {
	return func(s *Settings) error {
		s.config = c
		return nil
	}
}

func Repo(r *repo.FsRepo) Option {
	return func(s *Settings) error {
		s.repo = r
		return nil
	}
}

func IPFSClient(client ipfs.Client) Option {
	return func(s *Settings) error {
		s.options[new(ipfs.Client)] = fx.Supply(client)
		return nil
	}
}

func ComputeNode(enabled bool) Option {
	return func(s *Settings) error {
		if enabled {
			s.options[new(compute.ComputeNode)] = compute.Module
		}
		return nil
	}
}

func RequesterNode(enabled bool) Option {
	return func(s *Settings) error {
		if enabled {
			s.options[new(requester.RequesterNode)] = requester.Module
		}
		return nil
	}
}
