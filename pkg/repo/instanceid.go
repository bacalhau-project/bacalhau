package repo

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

// GenerateInstanceID creates a unique, anonymous identifier for the instance of bacalhau.
func GenerateInstanceID() string {
	// Get machine ID, which provides a consistent identifier for the device
	machineID, err := machineid.ID()
	if err != nil {
		// if we fail to read a machineID, generate a UUID
		machineID = uuid.NewString()
	}

	// hash the machineID to make it anonymous.
	hash := sha256.New()
	hash.Write([]byte(machineID))
	return hex.EncodeToString(hash.Sum(nil))
}
