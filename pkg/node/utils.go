package node

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

// TODO: this was copied from cli/serve.go, but doesn't seem right
func getTLSCertificate(cfg types.Bacalhau) (string, string, error) {
	cert := cfg.API.TLS.CertFile
	key := cfg.API.TLS.KeyFile
	if cert != "" && key != "" {
		return cert, key, nil
	}
	if cert != "" && key == "" {
		return "", "", fmt.Errorf("invalid config: TLS cert specified without corresponding private key")
	}
	if cert == "" && key != "" {
		return "", "", fmt.Errorf("invalid config: private key specified without corresponding TLS certificate")
	}
	if !cfg.API.TLS.SelfSigned {
		return "", "", nil
	}
	log.Info().Msg("Generating self-signed certificate")
	var err error
	// If the user has not specified a private key, use their client key
	if key == "" {
		key, err = cfg.UserKeyPath()
		if err != nil {
			return "", "", err
		}
	}
	certFile, err := os.CreateTemp(os.TempDir(), "bacalhau_cert_*.crt")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to create temporary server certificate")
	}
	defer closer.CloseWithLogOnError(certFile.Name(), certFile)

	var ips []net.IP = nil
	if ip := net.ParseIP(cfg.API.Host); ip != nil {
		ips = append(ips, ip)
	}

	if privKey, err := crypto.LoadPKCS1KeyFile(key); err != nil {
		return "", "", err
	} else if caCert, err := crypto.NewSelfSignedCertificate(privKey, false, ips); err != nil {
		return "", "", errors.Wrap(err, "failed to generate server certificate")
	} else if err = caCert.MarshalCertificate(certFile); err != nil {
		return "", "", errors.Wrap(err, "failed to write server certificate")
	}
	cert = certFile.Name()
	return cert, key, nil
}

// getAllocatedResources returns the resources allocated to the node.
func getAllocatedResources(ctx context.Context, cfg types.Bacalhau, executionsPath string) (models.Resources, error) {
	systemCapacity, err := system.NewPhysicalCapacityProvider(executionsPath).GetTotalCapacity(ctx)
	if err != nil {
		return models.Resources{}, fmt.Errorf("failed to determine total system capacity: %w", err)
	}
	allocatedResources, err := scaleCapacityByAllocation(systemCapacity, cfg.Compute.AllocatedCapacity)
	if err != nil {
		return models.Resources{}, err
	}
	return allocatedResources, nil
}

func scaleCapacityByAllocation(systemCapacity models.Resources, scaler types.ResourceScaler) (models.Resources, error) {
	// if the system capacity is zero we should fail as it means the compute node will be unable to accept any work.
	if systemCapacity.IsZero() {
		return models.Resources{}, fmt.Errorf("system capacity is zero")
	}

	// if allocated capacity scaler is zero, return the system capacity
	if scaler.IsZero() {
		return systemCapacity, nil
	}

	// scale the system resources based on the allocation
	allocatedCapacity, err := scaler.ToResource(systemCapacity)
	if err != nil {
		return models.Resources{}, fmt.Errorf("allocating system capacity: %w", err)
	}

	return *allocatedCapacity, nil
}

func logDebugIfContextCancelled(ctx context.Context, cleanupErr error, msg string) {
	if cleanupErr == nil {
		return
	}
	if !errors.Is(cleanupErr, context.Canceled) {
		log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to close " + msg)
	} else {
		log.Ctx(ctx).Debug().Err(cleanupErr).Msgf("Context canceled: %s", msg)
	}
}
