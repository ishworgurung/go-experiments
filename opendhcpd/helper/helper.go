package helper

import (
	"errors"
	"net"
)

// Localip returns the local ip address of the first
// global unicast network interface.
func Localip() (net.IP, error) {
	nifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, nif := range nifs {
		addrs, err := nif.Addrs()
		if err != nil {
			return nil, err
		}
		if nif.Flags&net.FlagUp == 0 ||
			nif.Flags&net.FlagLoopback != 0 {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsGlobalUnicast() {
				return ip, nil
			}
		}
	}
	return nil, errors.New("valid IP address not found for all network interfaces")
}
