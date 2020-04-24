package dhcpv4

const (
	internalLeaseTableSize       = 1024
	errFailParseStartIP          = "failed to parse start IP address"
	errFailParseDefaultGatewayIP = "failed to parse default gateway IP address"
	errFailParseDNSIP            = "failed to parse DNS IP address"
	errFailParseNetmaskIP        = "failed to parse netmask IP address"
	errNegativeLeaseSec          = "lease duration must be > 0 seconds. ideally, keep it above 7200 seconds"
	errInvalidDomainName         = "invalid domain name was provided. ideally, use non-unicode (for now) domain names"
	errInitialisationFailed      = "could not initialise DHCPv4 handler"
)
