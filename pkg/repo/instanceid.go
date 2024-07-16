package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/denisbrodbeck/machineid"
)

// GenerateInstanceID creates a unique, anonymous identifier for the instance of bacalhau.
// It combines the machine ID and MAC address to ensure uniqueness across different
// environments, including virtual machines and cloud instances.
//
// The function aims to produce consistent results on repeated calls on the same host,
// but there are scenarios where the result may differ:
//
//  1. Network Configuration Changes: If network interfaces are added, removed, or their
//     order changes, a different MAC address might be selected.
//  2. Virtual Environments: In VMs or containers, network configurations might change
//     between runs, affecting the selected MAC address.
//  3. MAC Address Randomization: Some systems implement MAC address randomization for
//     privacy, which could cause the function to return different results.
//  4. Hardware Changes: Adding or removing network adapters could alter the result.
//  5. System Updates: Major system updates might affect how machine IDs are generated
//     or how network interfaces are enumerated.
//
// The function uses SHA-256 hashing to maintain user anonymity by not exposing actual
// hardware identifiers. The resulting ID is a 64-character hexadecimal string.
func GenerateInstanceID() (string, error) {
	var elements []string

	// Get machine ID, which provides a consistent identifier for the device
	machineID, err := machineid.ID()
	if err != nil {
		return "", fmt.Errorf("failed to get machine ID: %w", err)
	}
	elements = append(elements, machineID)

	// Get MAC address to add another layer of uniqueness, especially useful in virtualized environments
	// or cloned disk images which share a machineID.
	macAddr, err := getMacAddress()
	if err != nil {
		return "", fmt.Errorf("failed to get MAC address: %w", err)
	}
	elements = append(elements, macAddr)

	// Combine all elements into a single string
	combined := strings.Join(elements, "|")

	// Generate SHA-256 hash of the combined string
	// SHA-256 is used for several reasons:
	// 1. It provides a fixed-length output (64 hexadecimal characters), regardless of input size
	// 2. It's a one-way function, ensuring the original machine ID and MAC address can't be reversed
	// 3. It helps maintain user anonymity by not exposing actual hardware identifiers
	hash := sha256.New()
	hash.Write([]byte(combined))
	installID := hex.EncodeToString(hash.Sum(nil))

	return installID, nil
}

// getMacAddress retrieves the MAC address of the first available non-loopback network interface.
// This function is used to add an additional layer of uniqueness to the instance ID.
//
// Note: The returned MAC address may vary if network configurations change or in
// virtualized environments. See GenerateInstanceID comments for more details on variability.
func getMacAddress() (string, error) {
	// Get all network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Iterate through interfaces to find a suitable one
	for _, iface := range interfaces {
		// Check if the interface is up and not a loopback
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			// Look for an IPv4 address
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						// Return the MAC address of the first suitable interface found
						return iface.HardwareAddr.String(), nil
					}
				}
			}
		}
	}

	// If no suitable interface is found, return an error
	return "", fmt.Errorf("no suitable MAC address found")
}
