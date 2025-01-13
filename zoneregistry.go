package zoneregistry

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/miekg/dns"
)

var (
	intervalDefault = uint32(60)
	ttlDefault      = uint32(300)
)

type ZoneRegistry struct {
	Next     plugin.Handler
	Zones    []string
	Peers    []string
	interval uint32
	ttl      uint32

	Fall fall.F
}

func newZoneRegistry() *ZoneRegistry {
	return &ZoneRegistry{
		interval: intervalDefault,
		ttl:      ttlDefault,
	}
}

// ServeDNS implements the plugin.Handler interface.
func (zr *ZoneRegistry) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Debug("Received response")
	log.Debug("Testing")

	// Call next plugin (if any).
	return plugin.NextOrFailure(zr.Name(), zr.Next, ctx, w, r)
}
func (zr *ZoneRegistry) Name() string { return pluginName }
