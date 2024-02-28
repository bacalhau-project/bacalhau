package nats

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/system"
)

var schemeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+-.]*://`)

const defaultScheme = "nats://"

// RoutesFromStr parses route URLs from a string
// e.g. "nats://localhost:4222,nats://localhost:4223"
func RoutesFromStr(routesStr string) ([]*url.URL, error) {
	routes := strings.Split(routesStr, ",")
	if len(routes) == 0 {
		return nil, nil
	}
	var routeUrls []*url.URL
	for _, r := range routes {
		r = strings.TrimSpace(r)
		if !schemeRegex.MatchString(r) {
			r = defaultScheme + r
		}
		u, err := url.Parse(r)
		if err != nil {
			return nil, err
		}
		routeUrls = append(routeUrls, u)
	}
	return routeUrls, nil
}

// RoutesFromSlice parses route URLs from a slice of strings
func RoutesFromSlice(routes []string) ([]*url.URL, error) {
	if len(routes) == 0 {
		return []*url.URL{}, nil
	}
	return RoutesFromStr(strings.Join(routes, ","))
}

// CreateAuthSecret will return a signed hash of the nodeID
// provided, for use as a secret for NATS authentication.
func CreateAuthSecret(nodeID string) (string, error) {
	var keySig string
	keySig, err := system.SignForClient([]byte(nodeID))
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(keySig))
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}
