package publicapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
)

func (apiServer *APIServer) id(res http.ResponseWriter, req *http.Request) {
	switch apiTransport := apiServer.Controller.GetTransport().(type) { //nolint:gocritic
	case *libp2p.LibP2PTransport:
		id, err := apiTransport.HostID(context.Background())
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
