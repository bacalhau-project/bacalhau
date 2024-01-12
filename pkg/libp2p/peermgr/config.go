package peermgr

import (
	"time"

	"github.com/benbjohnson/clock"
)

type config struct {
	runInterval      time.Duration
	bootstrapTimeout time.Duration
	clock            clock.Clock
}

type Option func(c *config) error

func WithRunInterval(d time.Duration) Option {
	return func(c *config) error {
		c.runInterval = d
		return nil
	}
}

func WithBootstrapTimeout(d time.Duration) Option {
	return func(c *config) error {
		c.bootstrapTimeout = d
		return nil
	}
}

func WithClock(clk clock.Clock) Option {
	return func(c *config) error {
		c.clock = clk
		return nil
	}
}
