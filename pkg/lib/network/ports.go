package network

import (
	"net"
)

// GetFreePort returns a single available port by asking the operating
// system to pick one for us. Luckily ports are not re-used so after asking
// for a port number, we attempt to create a tcp listener.
//
// Essentially the same code as https://github.com/phayes/freeport but we bind
// to 0.0.0.0 to ensure the port is free on all interfaces, and not just localhost.GetFreePort
// Ports must be unique for an address, not an entire system and so checking just localhost
// is not enough.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// GetFreePorts returns an array available ports by asking the operating
// system to pick one for us.
//
// Essentially the same code as https://github.com/phayes/freeport apart from
// the caveats described in GetFreePort.
func GetFreePorts(count int) ([]int, error) {
	ports := []int{}

	for i := 0; i < count; i++ {
		port, err := GetFreePort()
		if err != nil {
			return nil, err
		}
		ports = append(ports, port)
	}
	return ports, nil
}
