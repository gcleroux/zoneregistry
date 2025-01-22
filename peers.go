package zoneregistry

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	healthPortDefault = uint32(8080)
	intervalDefault   = uint32(60)
	timeoutDefault    = 2000 * time.Millisecond
)

type peer struct {
	Host       string
	Healthy    bool
	HealthPort uint32

	A    net.IP
	AAAA net.IP
}

func NewPeer(host string) (*peer, error) {
	p := &peer{
		Host:       host,
		Healthy:    false,
		HealthPort: healthPortDefault,
	}
	if err := p.resolveHost(); err != nil {
		return nil, err
	}
	return p, nil
}

// ResolveHost resolves a host to its IPv4 (A) and IPv6 (AAAA) addresses.
func (p *peer) resolveHost() error {
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
	peers []*peer
	mu    sync.RWMutex
	index int

	Interval uint32
	Timeout  time.Duration
}

func NewPeersTracker() *PeersTracker {
	pt := &PeersTracker{
		Interval: intervalDefault,
		Timeout:  timeoutDefault,
		index:    0,
	}
	return pt
}

func (pt *PeersTracker) GetHealthyPeers() []*peer {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	healthyPeers := make([]*peer, 0, len(pt.peers))
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
	rotated := make([]*peer, n)
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
func (pt *PeersTracker) AddPeer(peer *peer) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.peers = append(pt.peers, peer)
}

// RemovePeer safely removes a peer using in-place deletion
func (pt *PeersTracker) RemovePeer(peer *peer) {
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
	ticker := time.NewTicker(time.Duration(pt.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pt.mu.Lock()
		for _, p := range pt.peers {
			healthy := isHealthy(*p, pt.Timeout)
			if p.Healthy != healthy {
				log.Debugf("Peer %s changed state: Ready=%v\n", p.Host, healthy)
			}
			p.Healthy = healthy
		}
		pt.mu.Unlock()
	}
}

func isHealthy(p peer, timeout time.Duration) bool {
	client := &http.Client{
		Timeout: timeout,
	}
	peerURL := fmt.Sprintf("http://%s:%d/health", p.Host, p.HealthPort)
	resp, err := client.Get(peerURL)
	if err != nil {
		log.Debug(err)
		return false
	}

	log.Debugf("%s - %d", p.Host, resp.StatusCode)
	return resp.StatusCode == http.StatusOK
}
