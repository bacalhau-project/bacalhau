package licensing

import (
	"fmt"
	"time"
)

//nolint:lll
const (
	// FreeTierMaxNodes is the maximum number of nodes allowed in the free tier
	FreeTierMaxNodes = 5

	// License validation messages
	licenseMessageSkipped          = "License validation is taking a vacation today! 🏖️ (Validation is currently disabled)"
	licenseMessageValid            = "All good! Your license is valid until %s with %d nodes ready to go! 🎉 Keep on computing!"
	licenseMessageFreeTierValid    = "No license? No problem! You can use Bacalhau for free with up to %d nodes, or grab a license at https://cloud.expanso.io when you're ready to expand"
	licenseMessageFreeTierExceeded = "Wow, you're popular! You've got %d nodes, but the free tier only covers %d. Level up at https://cloud.expanso.io to connect your whole node family!"
	licenseMessageExpired          = "Oops! Your license expired on %s. You can renew it at https://cloud.expanso.io, or stick with the free tier (%d nodes max) for now"
	licenseMessageExceededNodes    = "Your nodes are having quite the party! You have %d nodes but your license only covers %d. " +
		"Upgrade your license at https://cloud.expanso.io for a bigger dance floor!"
)

// GetSkippedMessage returns the message for when license validation is skipped
func GetSkippedMessage() string {
	return licenseMessageSkipped
}

// GetValidMessage returns the message for when the license is valid
func GetValidMessage(expiryDate time.Time, licensedNodes int) string {
	return fmt.Sprintf(licenseMessageValid, expiryDate.Format("2006-01-02"), licensedNodes)
}

// GetNoLicenseMessage returns the message for when no license is found
func GetNoLicenseMessage() string {
	return fmt.Sprintf(licenseMessageFreeTierValid, FreeTierMaxNodes)
}

// GetFreeTierExceededMessage returns the message for when the free tier node limit is exceeded
func GetFreeTierExceededMessage(currentNodes int) string {
	return fmt.Sprintf(licenseMessageFreeTierExceeded, currentNodes, FreeTierMaxNodes)
}

// GetExpiredMessage returns the message for when the license has expired
func GetExpiredMessage(expiryDate time.Time) string {
	return fmt.Sprintf(licenseMessageExpired, expiryDate.Format("2006-01-02"), FreeTierMaxNodes)
}

// GetExceededNodesMessage returns the message for when the license node limit is exceeded
func GetExceededNodesMessage(currentNodes, licensedNodes int) string {
	return fmt.Sprintf(licenseMessageExceededNodes, currentNodes, licensedNodes)
}
