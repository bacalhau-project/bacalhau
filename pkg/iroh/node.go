package iroh

import (
	"os"

	iroh "github.com/n0-computer/iroh-ffi/iroh"
)

func New() (interface{}, error) {
	node, err := iroh.NewIrohNode(os.TempDir())
	if err != nil {
		return nil, err
	}

	doc, err := node.DocNew()
	if err != nil {
		return nil, err
	}
	doc.GetContentBytes()
	ticket, err := doc.Share()
	if err != nil {
		return nil, err
	}
	return doc, err
}

/*
my process for doing locally:
$ cd iroh-ffi
$ git checkout b5/dall-e-example-fixes
$ ./make_go.sh
$ cd ../iroh-examples/dall_e_worker
$ export LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}:/path/to/iroh-ffi/target/debug"
$ export CGO_LDFLAGS="-liroh -L /path/to/iroh-ffi/target/debug"
$ export OPENAI_API_KEY="your_secret_api_key"
$ go run main.go $IROH_TICKET

you can get an iroh ticket either from iroh.network (the "invite" button in the console), or by running iroh start locally, then in another terminal
$ iroh console
> doc create --switch
> doc share write
# ticket will output here

forrest — Today at 8:43 AM
what does the replace directive look like in you main.go? something like replace github.com/n0-computer/iroh-ffi => ../../n0-computer/iroh-ffi/go?
b5 — Today at 8:58 AM
yep exactly
mine:
replace github.com/n0-computer/iroh-ffi => ../iroh-ffi/go
I have iroh-ffi and iroh-examples as sibling directories

*/
