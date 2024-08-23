package types

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Validatable is an interface for types that can be validated
type Validatable interface {
	Validate() error
}

// validateFields runs validation on all fields that implement Validatable
// and includes the field name in the error message if validation fails.
// It also supports validation of arrays or slices of Validatable elements.
func validateFields(c interface{}) error {
	v := reflect.ValueOf(c)
	t := reflect.TypeOf(c)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		// Check if the field is a slice or array
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			// Iterate over the elements of the slice or array
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if elem.CanInterface() {
					if obj, ok := elem.Interface().(Validatable); ok {
						if err := obj.Validate(); err != nil {
							return fmt.Errorf("validation failed for field '%s' at index %d: %w", fieldName, j, err)
						}
					}
				}
			}
		} else {
			// Check if the field implements Validatable and call it.
			if field.CanInterface() {
				if obj, ok := field.Interface().(Validatable); ok {
					if err := obj.Validate(); err != nil {
						return fmt.Errorf("validation failed for field '%s': %w", fieldName, err)
					}
				}
			}
		}
	}
	return nil
}

func validateFileIffExists(path string) error {
	if path != "" {
		if info, err := os.Stat(path); err != nil {
			return fmt.Errorf("file at path %q cannot be read: %w", path, err)
		} else if info.IsDir() {
			return fmt.Errorf("path %q must be a file. receieved directory", path)
		}
	}
	return nil
}

// validateAddress validates an address in the form of "host:port"
func validateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Split the host and port
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("address %q must be a valid host:port pair: %w", address, err)
	}

	// Validate host
	if host == "" {
		return fmt.Errorf("address %q requires a valid host", address)
	}
	if host == "0.0.0.0" || host == "127.0.0.1" {
		// These hosts are usually valid for listening
	} else if ip := net.ParseIP(host); ip == nil {
		return fmt.Errorf("address %q has an invalid host: %q", address, host)
	}

	// Validate port
	if portStr == "" {
		return fmt.Errorf("address %q requires a port", address)
	}
	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		return fmt.Errorf("port must be a valid number: %w", err)
	}
	if port == 0 {
		return fmt.Errorf("address %q port cannot be 0", address)
	}

	return nil
}

// validateURL checks if the provided URL string is valid and meets specific criteria.
// It first ensures the URL is not empty, then parses it using url.Parse.
// The function verifies the presence of a host, and if valid schemas are provided,
// it checks that the URL's scheme is one of the acceptable ones. If no schemas are provided,
// then it passes any schema present in the URL. The function also validates that the port,
// if specified, is not 0 and falls within the valid range (1-65535). Detailed error messages
// are returned if any of these conditions are not met.
func validateURL(address string, validSchemas ...string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	parsedURL, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("address %q must be a valid URL: %w", address, err)
	}

	// Validate hostname
	if parsedURL.Hostname() == "" {
		return fmt.Errorf("address %q requires a valid hostname", address)
	}

	// Validate schemes if provided
	if len(validSchemas) > 0 {
		scheme := strings.ToLower(parsedURL.Scheme)
		if scheme == "" {
			return fmt.Errorf("address %q requires a scheme (e.g., http, https)", address)
		}

		isValidScheme := false
		for _, validSchema := range validSchemas {
			if scheme == strings.ToLower(validSchema) {
				isValidScheme = true
				break
			}
		}

		if !isValidScheme {
			return fmt.Errorf(
				"address %q requires a valid scheme. Must be one of: %v. Received: %q",
				address, validSchemas, parsedURL.Scheme,
			)
		}
	}

	// Validate port
	port := parsedURL.Port()
	if port == "0" {
		return fmt.Errorf("address %q cannot use port 0, as it is reserved and cannot be used", address)
	} else if port != "" {
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return fmt.Errorf("address %q contains an invalid port: %q", address, port)
		}
	}

	return nil
}
