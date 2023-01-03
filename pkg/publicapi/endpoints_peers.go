package publicapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
)

// peers godoc
//
//	@ID						peers
//	@Summary				Returns the peers connected to the host via the transport layer.
//	@Description.markdown	endpoints_peers
//	@Tags					Utils
//	@Produce				json
//	@Success				200	{object}	map[string][]string{}
//	@Failure				500	{object}	string
//	@Router					/peers [get]
func (apiServer *APIServer) peers(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/peers")
	defer span.End()

	// switch on apiTransport type to get the right method
	// we need to use a switch here because we want to look at .(type)
	// ^ that is a note for you gocritic
	switch apiTransport := apiServer.transport.(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		peers, err := apiTransport.GetPeers(ctx)
		if err != nil {
			http.Error(res, fmt.Sprintf("Error getting peers: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		// write response to res
		res.WriteHeader(http.StatusOK)
		err = json.NewEncoder(res).Encode(peers)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	http.Error(res, "Not a libp2p transport", http.StatusInternalServerError)
}
