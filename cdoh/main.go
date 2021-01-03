package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/babolivier/go-doh-client"
)

func main() {
	fqdnAddr := flag.String("fqdn", "", "-fqdn example.net")
	resolverAddr := flag.String("resolver", "doh.powerdns.org", "-resolver doh.powerdns.org")
	flag.Parse()
	if len(*fqdnAddr) == 0 || len(*resolverAddr) == 0 {
		flag.Usage()
		os.Exit(-1)
	}
	resolver := doh.Resolver{
		Host:  *resolverAddr,
		Class: doh.IN,
	}
	a4, ttl4, err := resolver.LookupA(*fqdnAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s. %d IN A %s\n", *fqdnAddr, ttl4, a4[0].IP4)

	a6, ttl6, err := resolver.LookupAAAA(*fqdnAddr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s. %d IN A %s\n", *fqdnAddr, ttl6, a6[0].IP6)
}
