package dhcpv4

import (
	"errors"
	"net"
	"strings"
	"time"
	"unicode"

	"github.com/ishworgurung/opendhcpd/helper"
	dhcp "github.com/krolaw/dhcp4"
	"github.com/rs/zerolog"
)

type Handler struct {
	ip            net.IP         // Server IP to use
	options       dhcp.Options   // Options to send to DHCP Clients
	start         net.IP         // Start of IP range to distribute
	leaseRange    int            // Number of IPs to distribute (starting from start)
	leaseDuration time.Duration  // Lease period
	leases        map[int]lease  // Map to keep track of leases
	logger        zerolog.Logger // The logger
	dhcpContex    dhcpCtx
}

func New(start, router, netmask, dns string, max, leaseSec int, domainName string, log zerolog.Logger) (*Handler, error) {
	var l, si, r, sm, d net.IP
	var err error

	if l, err = helper.Localip(); err != nil {
		log.Error().Msg(err.Error())
		return nil, err
	}
	l = l.To4()

	if si = net.ParseIP(start).To4(); si == nil {
		log.Error().Msg(errFailParseStartIP)
		return nil, errors.New(errFailParseStartIP)
	}
	if r = net.ParseIP(router).To4(); r == nil {
		log.Error().Msg(errFailParseDefaultGatewayIP)
		return nil, errors.New(errFailParseDefaultGatewayIP)
	}
	if d = net.ParseIP(dns).To4(); d == nil {
		log.Error().Msg(errFailParseDNSIP)
		return nil, errors.New(errFailParseDNSIP)
	}
	if sm = net.ParseIP(netmask).To4(); sm == nil {
		log.Error().Msg(errFailParseNetmaskIP)
		return nil, errors.New(errFailParseNetmaskIP)
	}
	if leaseSec <= 0 {
		log.Error().Msg(errNegativeLeaseSec)
		return nil, errors.New(errNegativeLeaseSec)
	}
	ld := time.Duration(leaseSec) * time.Second
	ml := int(max)
	dn := strings.TrimFunc(domainName, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	if len(dn) == 0 {
		log.Error().Msg(errInvalidDomainName)
		return nil, errors.New(errInvalidDomainName)
	}
	dd := &dhcpCtx{
		bindIP:        l,
		routerIP:      r,
		dnsIP:         d,
		subnetMask:    sm,
		startIP:       si,
		numLeaseMax:   ml,
		leaseDuration: ld,
		domainName:    dn,
	}
	dh, err := newDHCPv4Handler(dd, log)
	if err != nil {
		log.Error().Msgf("could not initialise DHCP dhcpv4 %s", err)
		return nil, errors.New(errInitialisationFailed)
	}
	return dh, nil
}
