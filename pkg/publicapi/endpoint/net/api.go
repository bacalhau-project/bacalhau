package net

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

type API struct {
	Host host.Host
}

func NewAPI(h host.Host) *API {
	return &API{Host: h}
}

func (a *API) RegisterRoutes(e *echo.Echo) {
	net := e.Group("/api/v1/net")
	net.GET("/peerss", a.peers)
	net.POST("/connect", a.connect)
	net.POST("/disconnect", a.disconnect)
	net.POST("/ping", a.ping)
	net.GET("/addresses", a.addrs)
}

func (a *API) addrs(e echo.Context) error {
	return e.JSON(http.StatusOK, &client.AddressesResponse{Addresses: a.Host.Addrs()})
}

func (a *API) peers(e echo.Context) error {
	var peerInfos []peer.AddrInfo
	for _, p := range a.Host.Peerstore().Peers() {
		peerInfos = append(peerInfos, a.Host.Peerstore().PeerInfo(p))
	}
	return e.JSON(http.StatusOK, &client.PeersResponse{Peers: peerInfos})
}

func (a *API) connect(e echo.Context) error {
	var req client.ConnectPeersRequest
	if err := e.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if swrm, ok := a.Host.Network().(*swarm.Swarm); ok {
		swrm.Backoff().Clear(req.Peer.ID)
	}

	if err := a.Host.Connect(e.Request().Context(), req.Peer); err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, client.ConnectPeersResponse{
		Success: true,
	})
}

func (a *API) disconnect(c echo.Context) error {
	var req client.DisconnectPeersRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := a.Host.Network().ClosePeer(req.Peer); err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, client.DisconnectPeersResponse{
		Success: true,
	})
}

func (a *API) ping(c echo.Context) error {
	var req client.PingPeerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, ok := <-ping.Ping(c.Request().Context(), a.Host, req.Peer)
	if !ok {
		return c.JSON(http.StatusInternalServerError, "no ping received")
	}
	msg := ""
	if result.Error != nil {
		msg = result.Error.Error()
	}
	return c.JSON(http.StatusOK, &client.PingPeerResponse{TTL: result.RTT, Msg: msg})
}
