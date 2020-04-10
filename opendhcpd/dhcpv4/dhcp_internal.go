package dhcpv4

import (
	"math/rand"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type lease struct {
	nic    string    // Client's CHAddr
	expiry time.Time // When the lease expires
}

// dhcpCtx holds the internal DHCP context
type dhcpCtx struct {
	bindIP        net.IP
	routerIP      net.IP
	dnsIP         net.IP
	subnetMask    net.IP
	startIP       net.IP
	numLeaseMax   int
	leaseDuration time.Duration
	domainName    string
}

// New creates a new DHCPv4 server object
func newDHCPv4Handler(dhcpIo *dhcpCtx, zlogger zerolog.Logger) (*Handler, error) {
	dhandler := &Handler{
		ip:            []byte(dhcpIo.bindIP),
		start:         dhcpIo.startIP,
		leaseDuration: dhcpIo.leaseDuration,
		leaseRange:    dhcpIo.numLeaseMax,
		leases:        make(map[int]lease, internalLeaseTableSize),
		options: dhcp.Options{
			dhcp.OptionSubnetMask:       []byte(dhcpIo.subnetMask),
			dhcp.OptionRouter:           []byte(dhcpIo.routerIP),
			dhcp.OptionDomainNameServer: []byte(dhcpIo.dnsIP),
			dhcp.OptionDomainName:       []byte(dhcpIo.domainName),
		},
		logger: zlogger,
	}
	return dhandler, nil
}

// Start starts serving DHCPv4 replies to DHCPv4 clients.
func (h *Handler) Start() error {
	h.logger.Info().Msgf("dhcpv4 server started listening on %s:67", h.ip)
	return dhcp.ListenAndServe(h)
}

// ServeDHCP serves the response to DHCPv4 clients based on the type of request they ask.
func (h *Handler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
	if reqIP == nil {
		reqIP = net.IP(p.CIAddr())
	}
	switch msgType {

	case dhcp.Discover:
		free, nic := -1, p.CHAddr().String()
		for i, v := range h.leases { // Find previous lease
			if v.nic == nic {
				free = i
				goto reply
			}
		}
		if free = h.freeLease(); free == -1 {
			return
		}
	reply:
		h.logger.Info().Msgf("Sent lease to %s", reqIP)
		return dhcp.ReplyPacket(p, dhcp.Offer, h.ip, dhcp.IPAdd(h.start, free), h.leaseDuration,
			h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))

	case dhcp.Request:
		if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.ip) {
			return nil // Message not for this dhcp server
		}

		if reqIP == nil {
			reqIP = net.IP(p.CIAddr())
		}
		if len(reqIP) == 4 && !reqIP.Equal(net.IPv4zero) {
			if leaseNum := dhcp.IPRange(h.start, reqIP) - 1; leaseNum >= 0 && leaseNum < h.leaseRange {
				if l, exists := h.leases[leaseNum]; !exists || l.nic == p.CHAddr().String() {
					h.leases[leaseNum] = lease{nic: p.CHAddr().String(), expiry: time.Now().Add(h.leaseDuration)}
					h.logger.Info().Msgf("Sent ACK to %s", reqIP)
					return dhcp.ReplyPacket(p, dhcp.ACK, h.ip, reqIP, h.leaseDuration,
						h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
				}
			}
			h.logger.Info().Msgf("Received DHCP request for invalid IP address %s", reqIP)
		}
		h.logger.Info().Msgf("Sent NAK to %s", reqIP)
		return dhcp.ReplyPacket(p, dhcp.NAK, h.ip, nil, 0, nil)

	case dhcp.Release, dhcp.Decline:
		nic := p.CHAddr().String()
		for i, v := range h.leases {
			if v.nic == nic {
				delete(h.leases, i)
				log.Printf("Deleted the lease %s", reqIP)
				break
			}
		}
	}
	return nil
}

func (h *Handler) freeLease() int {
	now := time.Now()
	b := rand.Intn(h.leaseRange) // Try random first
	for _, v := range [][]int{{b, h.leaseRange}, {0, b}} {
		for i := v[0]; i < v[1]; i++ {
			if l, ok := h.leases[i]; !ok || l.expiry.Before(now) {
				return i
			}
		}
	}
	return -1
}
