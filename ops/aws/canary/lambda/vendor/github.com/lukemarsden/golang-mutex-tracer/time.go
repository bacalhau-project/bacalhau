package muxtracer

import "time"

func now() uint64 {
	return uint64(time.Now().UnixNano())
}
