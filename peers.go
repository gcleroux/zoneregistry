package zoneregistry

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	roleDefault     = "primary"
	protocolDefault = "http"
	pathDefault     = "/health"
	portDefault     = uint32(8080)
)

type Peer struct {
	Host    string
	Role    string
	Healthy bool
	Labels  []string

	Protocol string
	Path     string
	Port     uint32

	IPv4 net.IP
	IPv6 net.IP
}

func NewPeer() *Peer {
	return &Peer{
		Role:     roleDefault,
		Protocol: protocolDefault,
		Path:     pathDefault,
		Port:     portDefault,
	}
}

func (p Peer) isHealthy(c *http.Client) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	urls := []string{}
	// Prioritize IPv6
	if p.IPv6 != nil {
		url := fmt.Sprintf("%s://[%s]:%d%s", p.Protocol, p.IPv6.String(), p.Port, p.Path)
		urls = append(urls, url)
	}
	if p.IPv4 != nil {
		url := fmt.Sprintf("%s://%s:%d%s", p.Protocol, p.IPv4.String(), p.Port, p.Path)
		urls = append(urls, url)
	}
	results := make(chan bool, len(urls))

	for _, url := range urls {
		go func(u string) {
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				log.Debugf("health check request creation failed for %s: %v", u, err)
				results <- false
				return
			}

			resp, err := c.Do(req)
			if err != nil {
				log.Debugf("Health check failed for %s: %v", u, err)
				results <- false
				return
			}
			defer resp.Body.Close()

			log.Debugf("%s - %d", u, resp.StatusCode)
			if resp.StatusCode == http.StatusOK {
				results <- true
				return
			}
			results <- false
		}(url)
	}

	for range urls {
		select {
		case success := <-results:
			if success {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
	return false
}
