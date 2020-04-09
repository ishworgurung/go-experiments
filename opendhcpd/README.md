# README

This is a no non-sense minimal DHCPv4 server 

## Usage summary

```bash
NAME:
   opendhcpd - no nonsense minimal DHCPv4 daemon

USAGE:
   opendhcpd [global options] command [command options] [arguments...]

VERSION:
   0.2.0

COMMANDS:
     run-server, rs  run opendhcpd
     help, h         Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version

```

## Run server options

Run in foreground:

```bash
$ ./opendhcpd rs -h
NAME:
   opendhcpd run-server - run opendhcpd

USAGE:
   opendhcpd run-server [command options] [arguments...]

OPTIONS:
   --dhcp_start value, -s value          dhcp start
   --dhcp_range value, -r value          dhcp range
   --default_gw value, -g value          dhcp gateway
   --dns_resolver value, -d value        dns resolver
   --subnet_mask value, -m value         subnet mask
   --lease_duration_sec value, -l value  lease duration in seconds
   --domain_name value, -n value         domain name
```

Run in background
```bash
$ ./opendhcpd bg -h
NAME:
   opendhcpd background - run opendhcpd in background

USAGE:
   opendhcpd background [command options] [arguments...]

OPTIONS:
   --dhcp_start value, -s value          dhcp start
   --dhcp_range value, -r value          dhcp range
   --default_gw value, -g value          dhcp gateway
   --dns_resolver value, -d value        dns resolver
   --subnet_mask value, -m value         subnet mask
   --lease_duration_sec value, -l value  lease duration in seconds
   --domain_name value, -n value         domain name
```

## Full options usage

```bash
$ sudo opendhcpd rs -s 10.10.200.10 -r 90 -g 10.10.200.1 -d 10.10.200.2 -m 255.255.255.0 -l 7200 -n foobar.local
9:08PM INF dhcpv4 server started listening on 10.10.200.5:67
9:08PM INF Sent ACK to 10.10.200.11
9:10PM INF Received DHCP request for invalid IP address 172.17.2.22
9:10PM INF Sent NAK to 172.17.2.22
```

Pull requests are most welcome. Thanks!

## Installation

### Using Git

```bash
$ git clone https://git.lsof.xyz/opendhcpd
$ cd opendhcpd && go build
```

### Using master branch tarball

```bash
$ curl -SLO https://git.lsof.xyz/git.lsof.xyz/opendhcpd/snapshot/opendhcpd-master.tar.gz
$ tar xf opendhcpd-master.tar.gz
$ cd opendhcpd-master && go build
```

## Credit

Big thanks to [github.com/krolaw/dhcp4](https://github.com/krolaw/dhcp4).

