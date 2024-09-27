package types

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

func CastConfigValueForKey(key string, value any) (any, error) {
	key = strings.ToLower(key)
	typ, ok := AllKeys()[key]
	if !ok {
		return nil, bacerrors.New("%q is not a valid config key", key).
			WithHint("Run '%s config list' for the complete list of valid config keys", os.Args[0]).
			WithCode(bacerrors.ValidationError).
			WithComponent("config")
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return parseString(key, v, typ)
	case []string:
		return parseStringSlice(key, v, typ)
	default:
		return nil, fmt.Errorf("DEVELOPER ERROR CastConfigValueForKey called with unsupported type: %T", v)
	}
}

func parseString(key, value string, typ reflect.Type) (any, error) {
	if typ == reflect.TypeOf(Duration(0)) {
		return parseDuration(key, value)
	}
	if typ == reflect.TypeOf([]string{}) {
		return parseStringToSlice(value)
	}
	return parseByKind(key, value, typ)
}

func parseStringToSlice(value string) ([]string, error) {
	// Check for invalid separators
	if strings.Contains(value, ";") || strings.Contains(value, " ") {
		return nil, bacerrors.New("invalid separator in string slice '%s', only comma (,) is allowed", value).
			WithCode(bacerrors.ValidationError).
			WithComponent("config")
	}

	// If there's no comma, return a slice with a single element
	if !strings.Contains(value, ",") {
		return []string{value}, nil
	}

	// Split the string by comma
	tokens := strings.Split(value, ",")

	// Check for empty tokens
	for i, token := range tokens {
		trimmedToken := strings.TrimSpace(token)
		if trimmedToken == "" {
			return nil, bacerrors.New("empty token found at position %d in string slice '%s'", i, value).
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}
		tokens[i] = trimmedToken // Store the trimmed token back
	}

	return tokens, nil
}

func parseStringSlice(key string, values []string, typ reflect.Type) (any, error) {
	if typ == reflect.TypeOf(Duration(0)) {
		return parseDuration(key, values[0])
	}
	if typ == reflect.TypeOf([]string{}) {
		// NB: for the case `config set` is ued like `config set compute.orchestrators=123.456.789,987.654.321`
		if len(values) == 1 && strings.Contains(values[0], ",") {
			return strings.Split(values[0], ","), nil
		}
		return values, nil
	}
	return parseByKind(key, values[0], typ)
}

func parseDuration(key, value string) (string, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return "", bacerrors.New("config key: %q expects a valid duration value, received: %q", key, value).
			WithHint("Accepted formats: 'h' (hours), 'm' (minutes), 's' (seconds), 'ms' (milliseconds), etc. " +
				"Example: '2h45m', '30s', '100ms").
			WithCode(bacerrors.ValidationError).
			WithComponent("config")
	}
	return duration.String(), nil
}

func parseByKind(key, value string, typ reflect.Type) (any, error) {
	switch typ.Kind() {
	case reflect.String:
		return value, nil
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return "", bacerrors.New("config key: %q expects a boolean value, received: %q", key, value).
				WithHint("Accepted formats: 'true' or 'false'").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}
		return parsed, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", bacerrors.New("config key: %q expects an integer value, received: %q", key, value).
				WithHint("Accepted formats: '1', '2', '10', etc.").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}
		return parsed, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return "", bacerrors.New("config key: %q expects a positive integer value, received: %q", key, value).
				WithHint("Accepted formats: '1', '2', '10', etc.").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}
		return parsed, nil
	case reflect.Map:
		tokens := strings.Split(value, ",")
		return StringSliceToMap(tokens)
	default:
		return nil, fmt.Errorf("parsing value: '%s' for key: '%s': unsupported configuration type: %v", value, key, typ)
	}
}

func StringSliceToMap(slice []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, item := range slice {
		tokens := strings.Split(item, "=")
		if len(tokens) < 2 {
			return nil, bacerrors.New("invalid format %s: expected 'key=value', but found no '=' in '%s'", slice, item).
				WithHint("Accepted formats: 'key=value', 'key1=value1,key2=value2', etc.").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}

		if len(tokens) > 2 {
			return nil, bacerrors.New("invalid format %s: found multiple '=' in '%s'. Only one '=' is allowed per key-value pair", slice, item).
				WithHint("Accepted formats: 'key=value', 'key1=value1,key2=value2', etc.").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}

		key := tokens[0]
		value := tokens[1]

		if key == "" {
			return nil, bacerrors.New("invalid format %s: missing key before '=' in '%s'. A valid key is required", slice, item).
				WithHint("Accepted formats: 'key=value', 'key1=value1,key2=value2', etc.").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}

		if value == "" {
			return nil, bacerrors.New("invalid format %s: missing value after '=' for key '%s'. A valid value is required", slice, key).
				WithHint("Accepted formats: 'key=value', 'key1=value1,key2=value2', etc.").
				WithCode(bacerrors.ValidationError).
				WithComponent("config")
		}

		result[key] = value
	}

	return result, nil
}

func AllKeys() map[string]reflect.Type {
	config := Bacalhau{}
	paths := make(map[string]reflect.Type)
	buildPathMap(reflect.ValueOf(config), "", paths)
	return paths
}

func buildPathMap(v reflect.Value, prefix string, paths map[string]reflect.Type) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			tag := field.Tag.Get("yaml")
			if tag == "" {
				tag = field.Name
			} else {
				tag = strings.Split(tag, ",")[0]
			}
			fieldPath := prefix + strings.ToLower(tag)
			buildPathMap(v.Field(i), fieldPath+".", paths)
		}
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Ptr:
		paths[prefix[:len(prefix)-1]] = v.Type()
	default:
		paths[prefix[:len(prefix)-1]] = v.Type()
	}
}
