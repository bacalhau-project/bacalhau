package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"

	"github.com/filecoin-project/bacalhau/pkg/system"
)

func (apiServer *APIServer) id(res http.ResponseWriter, req *http.Request) {
	_, span := system.GetSpanFromRequest(req, "apiServer/id")
	defer span.End()

	switch apiTransport := apiServer.transport.(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		id := apiTransport.HostID()
		res.WriteHeader(http.StatusOK)
		err := json.NewEncoder(res).Encode(id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}
