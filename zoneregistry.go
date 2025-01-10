package zoneregistry

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

type ZoneRegistry struct {
	Next plugin.Handler
}

func newZoneRegistry() *ZoneRegistry {
	return &ZoneRegistry{}
}

// ServeDNS implements the plugin.Handler interface.
func (zr *ZoneRegistry) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	log.Debug("Received response")

	// Call next plugin (if any).
	return plugin.NextOrFailure(zr.Name(), zr.Next, ctx, w, r)
}
func (zr *ZoneRegistry) Name() string { return pluginName }
