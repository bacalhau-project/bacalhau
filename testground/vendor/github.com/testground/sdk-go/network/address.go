package network

import (
	"fmt"
	"net"
)

// GetDataNetworkIP examines the local network interfaces, and tries to find our
// assigned IP within the data network.
//
// This function returns the IP and a nil error if found. If running in
// a sidecar-less environment, the error ErrNoTrafficShaping is returned.
func (c *Client) GetDataNetworkIP() (net.IP, error) {
	re := c.runenv
	if !re.TestSidecar {
		// this must be a local:exec runner and we currently don't support
		// traffic shaping on it for now, just return the loopback address
		return net.ParseIP("127.0.0.1"), nil
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get local network interfaces: %s", err)
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			re.RecordMessage("error getting addrs for interface: %s", err)
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				ip := v.IP.To4()
				if ip == nil {
					re.RecordMessage("ignoring non ip4 addr %s", v)
					continue
				}
				if re.TestSubnet.Contains(ip) {
					re.RecordMessage("detected data network IP: %s", v)
					return v.IP, nil
				} else {
					re.RecordMessage("%s not in data subnet %s, ignoring", ip, re.TestSubnet.String())
				}
			}
		}
	}
	return nil, fmt.Errorf("unable to determine data network IP. no interface found with IP in %s", re.TestSubnet.String())
}

// MustGetDataNetworkIP calls GetDataNetworkIP, and panics if it
// errors. It is suitable to use with runner.Invoke/InvokeMap, as long as
// this method is called from the main goroutine of the test plan.
func (c *Client) MustGetDataNetworkIP() net.IP {
	ip, err := c.GetDataNetworkIP()
	if err != nil {
		panic(err)
	}
	return ip
}
