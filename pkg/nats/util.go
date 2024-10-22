package nats

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
)

var schemeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+-.]*://`)

const defaultScheme = "nats://"

// RoutesFromStr parses route URLs from a string
// e.g. "nats://localhost:4222,nats://localhost:4223"
func RoutesFromStr(routesStr string, allowLocal bool) ([]*url.URL, error) {
	routes := strings.Split(routesStr, ",")
	if len(routes) == 0 {
		return nil, nil
	}

	var err error
	var routeUrls []*url.URL
	for _, r := range routes {
		r = strings.TrimSpace(r)
		if !schemeRegex.MatchString(r) {
			r = defaultScheme + r
		}
		u, err := url.Parse(r)
		if err != nil {
			return nil, NewConfigurationWrappedError(err, "invalid address: %s", routes)
		}
		routeUrls = append(routeUrls, u)
	}

	if !allowLocal {
		routeUrls, err = removeLocalAddresses(routeUrls)
		if err != nil {
			return nil, errors.Wrap(err, "failed to remove local addresses from NATS routes. please ensure settings do not contain a local address.") //nolint:lll
		}
	}

	return routeUrls, nil
}

// RoutesFromSlice parses route URLs from a slice of strings
func RoutesFromSlice(routes []string, allowLocal bool) ([]*url.URL, error) {
	if len(routes) == 0 {
		return []*url.URL{}, nil
	}
	return RoutesFromStr(strings.Join(routes, ","), allowLocal)
}

// removeLocalAddresses removes local addresses from a list of URLs
// and returns the result. This allows for accidental inclusion of local
// addresses in the list of NATS routes, even when we don't want to allow
// those local addresses (ie Jetstream clusters).
func removeLocalAddresses(routes []*url.URL) ([]*url.URL, error) {
	addrs, err := network.AllAddresses()
	if err != nil {
		return nil, NewConfigurationWrappedError(err, "invalid address: %s", routes)
	}

	localAddresses := lo.Map(addrs, func(item net.IP, _ int) string {
		return item.String()
	})

	result := make([]*url.URL, 0, len(routes))
	for _, u := range routes {
		if !slices.Contains(localAddresses, u.Hostname()) {
			result = append(result, u)
		}
	}
	return result, nil
}
