package node

import (
	"errors"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/samber/lo"
)

var supportedNetworks = []string{
	models.NetworkTypeLibp2p,
	models.NetworkTypeNATS,
}

type NetworkConfig struct {
	Type           string
	Libp2pHost     host.Host // only set if using libp2p transport, nil otherwise
	ReconnectDelay time.Duration

	// NATS config for requesters to be reachable by compute nodes
	Port              int
	AdvertisedAddress string
	Orchestrators     []string

	// NATS config for requester nodes to connect with each other
	ClusterName              string
	ClusterPort              int
	ClusterAdvertisedAddress string
	ClusterPeers             []string
}

func (c *NetworkConfig) Validate() error {
	var mErr *multierror.Error
	if validate.IsBlank(c.Type) {
		mErr = multierror.Append(mErr, errors.New("missing network type"))
	} else if !lo.Contains(supportedNetworks, c.Type) {
		mErr = multierror.Append(mErr, fmt.Errorf("network type %s not in supported values %s", c.Type, supportedNetworks))
	}
	return mErr.ErrorOrNil()
}
