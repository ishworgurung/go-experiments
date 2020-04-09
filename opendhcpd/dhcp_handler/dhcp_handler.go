package dhcpv4

import (
	"math/rand"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	internalLeaseTableSize = 1024
)

type dhcpV4Handler struct {
	ip            net.IP         // Server IP to use
	options       dhcp.Options   // Options to send to DHCP Clients
	start         net.IP         // Start of IP range to distribute
	leaseRange    int            // Number of IPs to distribute (starting from start)
	leaseDuration time.Duration  // Lease period
	leases        map[int]lease  // Map to keep track of leases
	logger        zerolog.Logger // The logger
}

type lease struct {
	nic    string    // Client's CHAddr
	expiry time.Time // When the lease expires
}

// DHCPContext is the contract between this module and module's user
type DHCPContext struct {
	BindIP        net.IP
	RouterIP      net.IP
	DnsIP         net.IP
	SubnetMask    net.IP
	StartIP       net.IP
	NumLeaseMax   int
	LeaseDuration time.Duration
	DomainName    string
}

// New creates a new DHCPv4 server object
func New(dhcpIo *DHCPContext, zlogger zerolog.Logger) (*dhcpV4Handler, error) {
	dhandler := &dhcpV4Handler{
		ip:            []byte(dhcpIo.BindIP),
		start:         dhcpIo.StartIP,
		leaseDuration: dhcpIo.LeaseDuration,
		leaseRange:    dhcpIo.NumLeaseMax,
		leases:        make(map[int]lease, internalLeaseTableSize),
		options: dhcp.Options{
			dhcp.OptionSubnetMask:       []byte(dhcpIo.SubnetMask),
			dhcp.OptionRouter:           []byte(dhcpIo.RouterIP),
			dhcp.OptionDomainNameServer: []byte(dhcpIo.DnsIP),
			dhcp.OptionDomainName:       []byte(dhcpIo.DomainName),
		},
		logger: zlogger,
	}
	return dhandler, nil
}

// Start starts serving DHCPv4 replies to DHCPv4 clients.
func (h *dhcpV4Handler) Start(dh dhcp.Handler) error {
	h.logger.Info().Msgf("dhcpv4 server started listening on %s:67", h.ip)
	return dhcp.ListenAndServe(dh)
}

// ServeDHCP serves the response to DHCPv4 clients based on the type of request they ask.
func (h *dhcpV4Handler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
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

func (h *dhcpV4Handler) freeLease() int {
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
