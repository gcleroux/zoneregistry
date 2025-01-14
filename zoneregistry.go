package zoneregistry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"
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

	subdomain := qname[:len(qname)-len(zone)-1]
	log.Debugf("Computed subdomain %s", subdomain)

	// Create the DNS response.
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	for _, peer := range zr.Peers {
		if !checkHealth(peer) {
			continue
		}
		cname := &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   qname,
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    zr.ttl,
			},
			Target: fmt.Sprintf("%s.%s", subdomain, peer),
		}
		msg.Answer = append(msg.Answer, cname)
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Errorf("Failed to send a response: %s", err)
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeSuccess, nil
}

func (zr *ZoneRegistry) Name() string { return pluginName }

func checkHealth(peer string) bool {
	client := &http.Client{
		Timeout: 200 * time.Millisecond,
	}
	peerURL := fmt.Sprintf("http://%s:8080/health", peer)
	resp, err := client.Get(peerURL)
	return err == nil && resp.StatusCode == http.StatusOK
}
