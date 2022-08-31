package model

import (
	"reflect"
	"strings"
	"unsafe"

	"github.com/rs/zerolog/log"
)

type KeyString string
type KeyInt int

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}

func PrintContextInternals(ctx interface{}, inner bool) {
	contextValues := reflect.ValueOf(ctx).Elem()
	contextKeys := reflect.TypeOf(ctx).Elem()

	if !inner {
		log.Debug().Msgf("\nFields for %s.%s\n", contextKeys.PkgPath(), contextKeys.Name())
	}

	if contextKeys.Kind() == reflect.Struct {
		for i := 0; i < contextValues.NumField(); i++ {
			reflectValue := contextValues.Field(i)
			reflectValue = reflect.NewAt(reflectValue.Type(), unsafe.Pointer(reflectValue.UnsafeAddr())).Elem()

			reflectField := contextKeys.Field(i)

			if reflectField.Name == "Context" {
				PrintContextInternals(reflectValue.Interface(), true)
			} else {
				log.Debug().Msgf("field name: %+v\n", reflectField.Name)
				log.Debug().Msgf("value: %+v\n", reflectValue.Interface())
			}
		}
	} else {
		log.Debug().Msgf("context is empty (int)\n")
	}
}
