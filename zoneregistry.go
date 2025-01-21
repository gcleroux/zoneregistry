package zoneregistry

import (
	"context"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

var ttlDefault = uint32(300)

type ZoneRegistry struct {
	Next  plugin.Handler
	Zones []string
	ttl   uint32
	Fall  fall.F

	Peers *PeersTracker
}

func newZoneRegistry() *ZoneRegistry {
	return &ZoneRegistry{
		ttl:   ttlDefault,
		Peers: NewPeersTracker(),
	}
}

// ServeDNS implements the plugin.Handler interface.
func (zr *ZoneRegistry) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	log.Debugf("Incoming query %s", state.QName())

	qname := state.QName()
	zone := plugin.Zones(zr.Zones).Matches(qname)
	if zone == "" {
		log.Debugf("Request %s has not matched any zones %v", qname, zr.Zones)
		return plugin.NextOrFailure(zr.Name(), zr.Next, ctx, w, r)
	}
	zone = qname[len(qname)-len(zone):] // maintain case of original query
	log.Debugf("Computed zone %s", zone)

	subdomain := strings.SplitN(qname, zone, 2)[0]
	log.Debugf("Computed subdomain %s", subdomain)

	// Create the DNS response.
	msg := new(dns.Msg)
	msg.SetReply(state.Req)
	msg.Authoritative = true

	for _, peer := range zr.Peers.GetHealthyPeers() {
		msg.Ns = append(msg.Ns, &dns.NS{Hdr: dns.RR_Header{Name: subdomain + peer.Host, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: zr.ttl}, Ns: peer.Host})

		if peer.A != nil {
			msg.Extra = append(msg.Extra, &dns.A{Hdr: dns.RR_Header{Name: peer.Host, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: zr.ttl}, A: peer.A})
		}
		if peer.AAAA != nil {
			msg.Extra = append(msg.Extra, &dns.AAAA{Hdr: dns.RR_Header{Name: peer.Host, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: zr.ttl}, AAAA: peer.AAAA})
		}
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Errorf("Failed to send a response: %s", err)
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeSuccess, nil
}

func (zr *ZoneRegistry) Name() string { return pluginName }
