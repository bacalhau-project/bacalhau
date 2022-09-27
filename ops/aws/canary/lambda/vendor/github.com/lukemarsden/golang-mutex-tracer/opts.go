package muxtracer

import "time"

type Opts struct {
	Threshold time.Duration
	Enabled   bool
	Id        string // use with
}
