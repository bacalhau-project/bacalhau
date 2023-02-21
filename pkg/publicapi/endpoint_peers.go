package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/libp2p/go-libp2p/core/peer"
)

// peers godoc
//
//	@ID						peers
//	@Summary				Returns the peers connected to the host via the transport layer.
//	@Description.markdown	endpoints_peers
//	@Tags					Utils
//	@Produce				json
//	@Success				200	{object}	[]peer.AddrInfo
//	@Failure				500	{object}	string
//	@Router					/peers [get]
func (apiServer *APIServer) peers(res http.ResponseWriter, _ *http.Request) {
	res.WriteHeader(http.StatusOK)
	var peerInfos []peer.AddrInfo
	for _, p := range apiServer.host.Peerstore().Peers() {
		peerInfos = append(peerInfos, apiServer.host.Peerstore().PeerInfo(p))
	}
	err := json.NewEncoder(res).Encode(peerInfos)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
