package main

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/reload"

	_ "github.com/gcleroux/zoneregistry"
)

func init() {
	dnsserver.Directives = append(dnsserver.Directives, "zoneregistry")
}

func main() {
	coremain.Run()
}
