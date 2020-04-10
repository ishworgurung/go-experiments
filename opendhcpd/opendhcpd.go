package main

import (
	"os"
	"strconv"

	"github.com/ishworgurung/opendhcpd/dhcpv4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sevlyar/go-daemon"
	"github.com/urfave/cli"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	app := cli.NewApp()
	app.Version = "0.2.0"
	app.Name = "opendhcpd"
	app.Usage = "no nonsense minimal DHCPv4 daemon"
	cliFlags := []cli.Flag{
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
			Flags:   cliFlags,
			Action: func(c *cli.Context) error {
				d, err := dhcpv4.New(
					c.String("dhcp_start"),
					c.String("default_gw"),
					c.String("subnet_mask"),
					c.String("dns_resolver"),
					c.Int("dhcp_range"),
					c.Int("lease_duration_sec"),
					c.String("domain_name"),
					log.Logger,
				)
				if err != nil {
					log.Fatal().Msg(err.Error())
				}
				log.Fatal().Msg(d.Start().Error())
				return nil
			},
		},
		{
			Name:    "background",
			Aliases: []string{"bg"},
			Usage:   "run opendhcpd in background",
			Flags:   cliFlags,
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
