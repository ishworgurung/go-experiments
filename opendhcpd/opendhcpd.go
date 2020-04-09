package main

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	dhcpv4 "github.com/ishworgurung/opendhcpd/dhcp_handler"
	"github.com/ishworgurung/opendhcpd/helper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli"
)

const (
	errFailParseStartIP          = "failed to parse start IP address"
	errFailParseDefaultGatewayIP = "failed to parse default gateway IP address"
	errFailParseDNSIP            = "failed to parse DNS IP address"
	errFailParseNetmaskIP        = "failed to parse netmask IP address"
	errNegativeLeaseSec          = "lease duration must be > 0 seconds. ideally, keep it above 7200 seconds"
	errInvalidDomainName         = "invalid domain name was provided. ideally, use non-unicode (for now) domain names"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	app := cli.NewApp()
	app.Version = "0.2.0"
	app.Name = "opendhcpd"
	app.Usage = "no nonsense minimal DHCPv4 daemon"
	cliFlag := []cli.Flag{
		cli.StringFlag{
			Name:  "dhcp_start,s",
			Usage: "dhcp start",
		},
		cli.StringFlag{
			Name:  "dhcp_range,r",
			Usage: "dhcp range",
		},
		cli.StringFlag{
			Name:  "default_gw,g",
			Usage: "dhcp gateway",
		},
		cli.StringFlag{
			Name:  "dns_resolver,d",
			Usage: "dns resolver",
		},
		cli.StringFlag{
			Name:  "subnet_mask,m",
			Usage: "subnet mask",
		},
		cli.StringFlag{
			Name:  "lease_duration_sec,l",
			Usage: "lease duration in seconds",
		},
		cli.StringFlag{
			Name:  "domain_name,n",
			Usage: "domain name",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "run-server",
			Aliases: []string{"rs"},
			Usage:   "run opendhcpd in foreground",
			Flags:   cliFlag,
			Action: func(c *cli.Context) error {
				d, err := newDHCPV4(
					c.String("dhcp_start"),
					c.String("default_gw"),
					c.String("subnet_mask"),
					c.String("dns_resolver"),
					c.Int("dhcp_range"),
					c.Int("lease_duration_sec"),
					c.String("domain_name"),
				)
				if d == nil {
					log.Fatal().Msg("failed to initialise dhcpv4 internal object. Did you pass all options?")
				}
				if err != nil {
					log.Fatal().Msg(err.Error())
				}
				dh, err := dhcpv4.New(d, log.Logger)
				if err != nil {
					log.Fatal().Msgf("could not initialise DHCP handler %s", err)
				}
				log.Fatal().Msg(dh.Start(dh).Error())
				return nil
			},
		},
		{
			Name:    "background",
			Aliases: []string{"bg"},
			Usage:   "run opendhcpd in background",
			Flags:   cliFlag,
			Action: func(c *cli.Context) error {
				s := c.String("dhcp_start")
				g := c.String("default_gw")
				m := c.String("subnet_mask")
				d := c.String("dns_resolver")
				r := c.Int("dhcp_range")
				l := c.Int("lease_duration_sec")
				n := c.String("domain_name")

				daemonCtx := &daemon.Context{
					PidFileName: "/var/run/opendhcpd.pid",
					PidFilePerm: 0644,
					LogFileName: "/var/log/opendhcpd.log",
					LogFilePerm: 0644,
					WorkDir:     "/tmp",
					Umask:       0022,
					Args: []string{
						"opendhcpd",
						"rs", "-s", s, "-r", strconv.Itoa(r), "-g", g,
						"-d", d, "-m", m, "-l", strconv.Itoa(l), "-n", n,
					},
				}
				reb, err := daemonCtx.Reborn()
				if err != nil {
					log.Fatal().Msgf("unable to daemonize opendhcpd: %s\n", err)
				}
				if reb != nil {
					log.Info().Msgf("failed %v\n", reb)
					return nil
				}
				defer daemonCtx.Release()
				log.Info().Msg("opendhcpd daemon started")
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func newDHCPV4(start, router, netmask, dns string, max int, leaseSec int, domainName string) (*dhcpv4.DHCPContext, error) {
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
	return &dhcpv4.DHCPContext{
		BindIP:        l,
		RouterIP:      r,
		DnsIP:         d,
		SubnetMask:    sm,
		StartIP:       si,
		NumLeaseMax:   ml,
		LeaseDuration: ld,
		DomainName:    dn,
	}, nil
}
