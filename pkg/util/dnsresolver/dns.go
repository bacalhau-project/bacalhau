package dnsresolver

import (
	"context"
	"net"
	"time"
)

var timeToWait = 2 * time.Second

func IsOnline(ctx context.Context, n string) bool {
	_, err := LookupIP(ctx, n, int(timeToWait))
	return err == nil
}

func LookupIP(ctx context.Context, n string, timeout int) ([]string, error) {
	r := net.DefaultResolver
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	return r.LookupHost(ctx, n)
}
