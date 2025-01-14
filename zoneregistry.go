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
	timeoutDefault  = 200 * time.Millisecond
)

type ZoneRegistry struct {
	Next     plugin.Handler
	Zones    []string
	Peers    map[string]bool
	interval uint32
	ttl      uint32
	timeout  time.Duration

	Fall fall.F
}

func newZoneRegistry() *ZoneRegistry {
	return &ZoneRegistry{
		Peers:    map[string]bool{},
		interval: intervalDefault,
		ttl:      ttlDefault,
		timeout:  timeoutDefault,
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

	// Look for healthy peers
	hasHealthyPeers := false
	for _, ok := range zr.Peers {
		if ok {
			hasHealthyPeers = true
			break
		}
	}

	for peer, ok := range zr.Peers {
		// Skip unhealthy peers if there are healthy ones
		if hasHealthyPeers && !ok {
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

func (zr *ZoneRegistry) HealthCheck() {
	ticker := time.NewTicker(time.Duration(zr.interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Debug("Performing healthchecks")
		for peer := range zr.Peers {
			zr.Peers[peer] = zr.isHealthy(peer)
		}
	}
}

func (zr *ZoneRegistry) isHealthy(peer string) bool {
	client := &http.Client{
		Timeout: zr.timeout,
	}
	peerURL := fmt.Sprintf("http://%s:8080/health", peer)
	resp, err := client.Get(peerURL)
	if err != nil {
		log.Debug(err)
		return false
	}

	log.Debugf("%s - %d", peer, resp.StatusCode)
	return resp.StatusCode == http.StatusOK
}
