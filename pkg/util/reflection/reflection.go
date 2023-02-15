package reflection

import (
	"fmt"
	"reflect"
	"strings"
)

func StructName(a any) string {
	delegateType := reflect.Indirect(reflect.ValueOf(a)).Type()
	path := strings.TrimPrefix(delegateType.PkgPath(), "github.com/filecoin-project/bacalhau/")
	if path == "" {
		return delegateType.Name()
	}

	name := fmt.Sprintf("%s.%s", path, delegateType.Name())
	return name
}
