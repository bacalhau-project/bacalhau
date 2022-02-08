// THIS FILE IS NOT FOR PRODUCTION USE OR INCLUSION IN ANY PACKAGE
// It is a convient place to add libraries from the rest of the

package bacalhau

import (
	"crypto/rand"
	"fmt"
	"math/big"
)


func bacalhau() {
	// ...
	r, _ := rand.Int(rand.Reader, big.NewInt(10))
	fmt.Printf("Test: %v", r)
}
