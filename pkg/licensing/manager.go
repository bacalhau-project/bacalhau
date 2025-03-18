package licensing

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/license"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
)

const (
	// validationInterval is the interval at which the license validation loop runs
	validationInterval = 1 * time.Hour
)

type ManagerParams struct {
	Reader             Reader
	NodesTracker       nodes.Tracker
	ValidationInterval time.Duration
	SkipValidation     bool
}

// manager handles license management and validation for the node
type manager struct {
	reader             Reader
	nodesLookup        nodes.Tracker
	validationInterval time.Duration
	skipValidation     bool
	stopChan           chan struct{}
	running            bool
	mu                 sync.Mutex
}

// NewManager creates and initializes a new manager
func NewManager(params ManagerParams) (Manager, error) {
	err := errors.Join(
		validate.NotNil(params.Reader, "license reader cannot be nil"),
		validate.NotNil(params.NodesTracker, "node lookup cannot be nil"),
	)
	if err != nil {
		return nil, err
	}

	if params.ValidationInterval == 0 {
		params.ValidationInterval = validationInterval
	}

	mngr := &manager{
		reader:             params.Reader,
		nodesLookup:        params.NodesTracker,
		validationInterval: params.ValidationInterval,
		skipValidation:     params.SkipValidation,
	}
	mngr.logLicenseState(true)
	return mngr, nil
}

// Start starts a background routine that periodically validates the license and logs warnings
func (l *manager) Start() {
	if l.skipValidation {
		return
	}

	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return
	}
	l.stopChan = make(chan struct{})
	l.running = true
	l.mu.Unlock()

	go func() {
		ticker := time.NewTicker(l.validationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-l.stopChan:
				l.mu.Lock()
				l.running = false
				l.mu.Unlock()
				return
			case <-ticker.C:
				l.logLicenseState(false)
			}
		}
	}()
}

func (l *manager) logLicenseState(logValid bool) {
	state := l.Validate()
	switch {
	case state.Type == LicenseValidationTypeSkipped:
		log.Debug().Msg(state.Message)
	case state.Type.IsValid():
		if logValid {
			log.Info().Msg(state.Message)
		}
	default:
		log.Warn().Msg(state.Message)
	}
}

// Stop stops the validation loop
func (l *manager) Stop() {
	l.mu.Lock()
	if !l.running {
		l.mu.Unlock()
		return
	}
	close(l.stopChan)
	l.running = false
	l.mu.Unlock()
}

// License returns the current license claims
func (l *manager) License() *license.LicenseClaims {
	return l.reader.License()
}

// Validate checks the current license state and returns a validation state
func (l *manager) Validate() LicenseValidationState {
	if l.skipValidation {
		return LicenseValidationState{
			Type:    LicenseValidationTypeSkipped,
			Message: GetSkippedMessage(),
		}
	}

	connectedNodes := l.nodesLookup.GetConnectedNodesCount()

	// Check if we have a license
	claims := l.License()
	if claims == nil {
		// No license configured, check if we're within free tier
		if connectedNodes <= FreeTierMaxNodes {
			return LicenseValidationState{
				Type:    LicenseValidationTypeFreeTierValid,
				Message: GetNoLicenseMessage(),
			}
		}
		return LicenseValidationState{
			Type:    LicenseValidationTypeFreeTierExceeded,
			Message: GetFreeTierExceededMessage(connectedNodes),
		}
	}

	// Check if license is expired
	if claims.IsExpired() {
		return LicenseValidationState{
			Type:    LicenseValidationTypeExpired,
			Message: GetExpiredMessage(claims.ExpiresAt.Time),
		}
	}

	// Check node limit
	maxNodes := claims.MaxNumberOfNodes()
	if maxNodes > 0 && connectedNodes > maxNodes {
		return LicenseValidationState{
			Type:    LicenseValidationTypeExceededNodes,
			Message: GetExceededNodesMessage(connectedNodes, maxNodes),
		}
	}

	return LicenseValidationState{
		Type:    LicenseValidationTypeValid,
		Message: GetValidMessage(claims.ExpiresAt.Time, maxNodes),
	}
}

// compile time check for interface implementation
var _ Manager = &manager{}
