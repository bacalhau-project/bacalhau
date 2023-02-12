package v1beta1

import "fmt"

// A Millicore represents a thousandth of a CPU core, which is a unit of measure
// used by Kubernetes. See also https://github.com/BTBurke/k8sresource.
type Millicores int

const (
	Millicore Millicores = 1
	Core      Millicores = 1000
)

const suffix string = "m"

// String returns a string representation of this Millicore, which is either an
// integer if this Millicore represents a whole number of cores or the number of
// Millicores suffixed with "m".
func (m Millicores) String() string {
	if m%Core == 0 {
		return fmt.Sprintf("%d", m/Core)
	} else {
		return fmt.Sprintf("%d%s", m, suffix)
	}
}
