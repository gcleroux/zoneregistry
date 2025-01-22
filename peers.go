package zoneregistry

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin"
)

var (
	healthPortDefault = uint32(8080)
	intervalDefault   = uint32(60)
	timeoutDefault    = uint32(5)
)

type Peer struct {
	Host       string
	Healthy    bool
	HealthPort uint32

	A    net.IP
	AAAA net.IP
}

func NewPeer(host string) *Peer {
	if h := plugin.Host(host).NormalizeExact(); len(h) != 0 {
		return &Peer{
			Host:       h[0],
			Healthy:    false,
			HealthPort: healthPortDefault,
		}
	}
	return nil
}

// ResolveHost resolves a host to its IPv4 (A) and IPv6 (AAAA) addresses.
func (p *Peer) resolveHost() error {
	ips, err := net.LookupIP(p.Host)
	if err != nil {
		return err
	}

	for _, ip := range ips {
		log.Debugf("Resolved host %s to %s", p.Host, ip.String())
		if ipv4 := ip.To4(); ipv4 != nil {
			p.A = ipv4
		} else {
			p.AAAA = ip
		}
	}
	return nil
}

type PeersTracker struct {
	peers []*Peer
	mu    sync.RWMutex
	index int

	Interval uint32
	Timeout  uint32
}

func NewPeersTracker() *PeersTracker {
	pt := &PeersTracker{
		Interval: intervalDefault,
		Timeout:  timeoutDefault,
		index:    0,
	}
	return pt
}

func (pt *PeersTracker) GetHealthyPeers() []*Peer {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	healthyPeers := make([]*Peer, 0, len(pt.peers))
	for _, peer := range pt.peers {
		if peer.Healthy {
			healthyPeers = append(healthyPeers, peer)
		}
	}
	// Return all peers if none are healthy
	if len(healthyPeers) == 0 {
		log.Debugf("No healthy peers found, returning all peers")
		healthyPeers = append(healthyPeers, pt.peers...)
	}

	// Rotate the list based on the round-robin index
	n := len(healthyPeers)
	rotated := make([]*Peer, n)
	if pt.index >= n {
		pt.index = 0 // Reset index to prevent OOB errors
	}

	// Rotate the list based on the round-robin index
	copy(rotated, healthyPeers[pt.index:])
	copy(rotated[n-pt.index:], healthyPeers[:pt.index])

	pt.index = (pt.index + 1) % n

	return rotated
}

// AddPeer safely adds or updates a peer
func (pt *PeersTracker) AddPeer(peer *Peer) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.peers = append(pt.peers, peer)
}

// RemovePeer safely removes a peer using in-place deletion
func (pt *PeersTracker) RemovePeer(peer *Peer) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Two-pointer technique for in-place removal
	i := 0
	for _, p := range pt.peers {
		if p.Host != peer.Host {
			pt.peers[i] = p
			i++
		}
	}
	pt.peers = pt.peers[:i]
}

func (pt *PeersTracker) StartHealthChecks() {
	var wg sync.WaitGroup
	ticker := time.NewTicker(time.Duration(pt.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pt.mu.Lock()
		for _, p := range pt.peers {
			wg.Add(1)
			go func(p *Peer) {
				defer wg.Done()
				healthy := isHealthy(*p, pt.Timeout)
				if p.Healthy != healthy {
					log.Debugf("Peer %s changed state: Ready=%v\n", p.Host, healthy)
				}
				p.Healthy = healthy
			}(p)
		}
		wg.Wait()
		pt.mu.Unlock()
	}
}

func isHealthy(p Peer, timeout uint32) bool {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d/health", p.Host, p.HealthPort), nil)
	if err != nil {
		log.Debugf("Health check request creation failed for %s: %v", p.Host, err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Debugf("Health check failed for %s: %v", p.Host, err)
		return false
	}
	defer resp.Body.Close()

	log.Debugf("%s - %d", p.Host, resp.StatusCode)
	return resp.StatusCode == http.StatusOK
}
