package zoneregistry

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

var (
	ttlDefault      = uint32(300)
	intervalDefault = uint32(60)
	timeoutDefault  = uint32(5)
)

type ZoneRegistry struct {
	Next     plugin.Handler
	Zones    []string
	TTL      uint32
	Interval uint32
	Timeout  uint32
	Fall     fall.F

	Peers []*Peer
	mu    sync.RWMutex
	index int
}

func newZoneRegistry() *ZoneRegistry {
	return &ZoneRegistry{
		TTL:      ttlDefault,
		Interval: intervalDefault,
		Timeout:  timeoutDefault,
	}
}

// ServeDNS implements the plugin.Handler interface.
func (zr *ZoneRegistry) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	start := time.Now()
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

	healthyPeers := zr.GetHealthyPeers()

	// Rotate the list based on the round-robin index
	n := len(healthyPeers)
	lbPeers := make([]*Peer, n)
	if zr.index >= n {
		zr.index = 0 // Reset index to prevent OOB errors
	}
	copy(lbPeers, healthyPeers[zr.index:])
	copy(lbPeers[n-zr.index:], healthyPeers[:zr.index])
	zr.index = (zr.index + 1) % n

	for _, peer := range lbPeers {
		msg.Ns = append(msg.Ns, &dns.NS{Hdr: dns.RR_Header{Name: subdomain + peer.Host, Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: zr.TTL}, Ns: peer.Host})

		if peer.IPv4 != nil {
			msg.Extra = append(msg.Extra, &dns.A{Hdr: dns.RR_Header{Name: peer.Host, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: zr.TTL}, A: peer.IPv4})
		}
		if peer.IPv6 != nil {
			msg.Extra = append(msg.Extra, &dns.AAAA{Hdr: dns.RR_Header{Name: peer.Host, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: zr.TTL}, AAAA: peer.IPv6})
		}
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Errorf("Failed to send a response: %s", err)
		return dns.RcodeServerFailure, err
	}
	queryCount.WithLabelValues(metrics.WithServer(ctx), zone).Inc()
	responseDuration.WithLabelValues(metrics.WithServer(ctx), zone).Observe(float64(time.Since(start).Seconds()))

	return dns.RcodeSuccess, nil
}

func (zr *ZoneRegistry) Name() string { return pluginName }

func (zr *ZoneRegistry) GetHealthyPeers() []*Peer {
	zr.mu.RLock()
	defer zr.mu.RUnlock()

	healthyPrimaryPeers := make([]*Peer, 0, len(zr.Peers))
	healthySecondaryPeers := make([]*Peer, 0, len(zr.Peers))
	for _, peer := range zr.Peers {
		if peer.Healthy && peer.Role == "primary" {
			healthyPrimaryPeers = append(healthyPrimaryPeers, peer)
		}
		if peer.Healthy && peer.Role == "secondary" {
			healthySecondaryPeers = append(healthySecondaryPeers, peer)
		}
	}

	if len(healthyPrimaryPeers) > 0 {
		return healthyPrimaryPeers
	}
	if len(healthySecondaryPeers) > 0 {
		return healthySecondaryPeers
	}

	// Return all peers if none are healthy
	log.Debugf("No healthy peers found, returning all peers")
	healthyPrimaryPeers = append(healthyPrimaryPeers, zr.Peers...)
	return healthyPrimaryPeers
}

func (zr *ZoneRegistry) StartHealthChecks() {
	var wg sync.WaitGroup
	ticker := time.NewTicker(time.Duration(zr.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		zr.mu.Lock()
		hp_count := atomic.Uint32{}
		hs_count := atomic.Uint32{}
		for _, p := range zr.Peers {
			wg.Add(1)
			go func(p *Peer) {
				defer wg.Done()
				client := &http.Client{
					Timeout: time.Duration(zr.Timeout) * time.Second,
				}

				status := p.isHealthy(client)
				if p.Healthy != status {
					log.Debugf("Peer %s changed state: Ready=%v\n", p.Host, status)
				}
				p.Healthy = status

				// Track peer count for metrics
				if status == true {
					if p.Role == "primary" {
						hp_count.Add(1)
					} else {
						hs_count.Add(1)
					}
				}
			}(p)
		}
		wg.Wait()
		zr.mu.Unlock()

		healthyPeers.WithLabelValues("primary").Set(float64(hp_count.Load()))
		healthyPeers.WithLabelValues("secondary").Set(float64(hs_count.Load()))
		unhealthyPeers.WithLabelValues("primary").Set(float64(len(zr.Peers) - int(hp_count.Load())))
		unhealthyPeers.WithLabelValues("secondary").Set(float64(len(zr.Peers) - int(hs_count.Load())))
	}
}
