package publicapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
)

func (apiServer *APIServer) id(res http.ResponseWriter, req *http.Request) {
	ctx, span := system.GetSpanFromRequest(req, "apiServer/id")
	defer span.End()

	switch apiTransport := apiServer.Controller.GetTransport().(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		id, err := apiTransport.HostID(ctx)
		if err != nil {
			http.Error(res, fmt.Sprintf("Error getting id: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
		err = json.NewEncoder(res).Encode(id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
}
