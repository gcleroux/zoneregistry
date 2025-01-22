package zoneregistry

import (
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/fall"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		input               string
		shouldErr           bool
		expectedZone        string
		expectedZones       int
		expectedPeers       []Peer
		expectedTTL         uint32
		expectedInterval    uint32
		expectedTimeout     uint32
		expectedFallthrough *fall.F
	}{
		// Validation tests
		{
			input:               `zoneregistry`,
			shouldErr:           false,
			expectedZone:        "",
			expectedZones:       1,
			expectedPeers:       nil,
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: nil,
		},
		{
			input:               `zoneregistry example.org`,
			shouldErr:           false,
			expectedZone:        "example.org.",
			expectedZones:       1,
			expectedPeers:       nil,
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: nil,
		},
		{
			input: `zoneregistry example.org {
						peers example.org example.com
						ttl 100
						interval 20
						timeout 10
					}`,
			shouldErr:     false,
			expectedZone:  "example.org.",
			expectedZones: 1,
			expectedPeers: []Peer{
				{Host: "example.org."},
				{Host: "example.com."},
			},
			expectedTTL:         100,
			expectedInterval:    20,
			expectedTimeout:     10,
			expectedFallthrough: nil,
		},
		{
			input: `zoneregistry example.org {
						fallthrough
					}`,
			shouldErr:           false,
			expectedZone:        "example.org.",
			expectedZones:       1,
			expectedPeers:       nil,
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: &fall.F{Zones: []string{"."}},
		},
		{
			input: `zoneregistry example.org {
						fallthrough example.com .
					}`,
			shouldErr:           false,
			expectedZone:        "example.org.",
			expectedZones:       1,
			expectedPeers:       nil,
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: &fall.F{Zones: []string{"example.com.", "."}},
		},
		// Error tests
		{
			input: `zoneregistry example.org {
						peers unresolvable.fake-tld
					}`,
			shouldErr:           true,
			expectedZone:        "example.org.",
			expectedZones:       1,
			expectedPeers:       nil,
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: nil,
		},
		{
			input: `zoneregistry example.org {
						ttl string_not_uint32
					}`,
			shouldErr:           true,
			expectedZone:        "example.org.",
			expectedZones:       1,
			expectedPeers:       nil,
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: nil,
		},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		zr, err := parse(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found %s for input %s", i, err, test.input)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
			}
		}

		// Validate zones
		if !test.shouldErr && test.expectedZone != "" {
			if test.expectedZones != len(zr.Zones) {
				t.Errorf("Test %d, expected %d zones for input %s, got: %d", i, test.expectedZones, test.input, len(zr.Zones))
			}
			if zr.Zones[0] != test.expectedZone {
				t.Errorf("Test %d, expected zone %q for input %s, got: %q", i, test.expectedZone, test.input, zr.Zones[0])
			}
		}
		// Validate Peers
		if test.expectedPeers != nil && !test.shouldErr {
			if len(test.expectedPeers) != len(zr.Peers.peers) {
				t.Errorf("Test %d, expected %d peers, got: %d", i, len(test.expectedPeers), len(zr.Peers.peers))
			}
			for j, peer := range test.expectedPeers {
				if zr.Peers.peers[j].Host != peer.Host {
					t.Errorf("Test %d, expected peer %q, got: %q", i, peer.Host, zr.Peers.peers[j].Host)
				}
			}
		}
		// Validate TTL
		if !test.shouldErr {
			if zr.ttl != test.expectedTTL {
				t.Errorf("Test %d, expected TTL %d, got: %d", i, test.expectedTTL, zr.ttl)
			}
		}
		// Validate Interval
		if !test.shouldErr {
			if zr.Peers.Interval != test.expectedInterval {
				t.Errorf("Test %d, expected INTERVAL %d, got: %d", i, test.expectedInterval, zr.Peers.Interval)
			}
		}
		// Validate Timeout
		if !test.shouldErr {
			if zr.Peers.Timeout != test.expectedTimeout {
				t.Errorf("Test %d, expected TIMEOUT %d, got: %d", i, test.expectedTimeout, zr.Peers.Timeout)
			}
		}
		// Validate Fallthrough
		if test.expectedFallthrough != nil && !test.shouldErr {
			if len(test.expectedFallthrough.Zones) != len(zr.Fall.Zones) {
				t.Errorf("Test %d, expected fallthrough zones %v, got: %v", i, test.expectedFallthrough.Zones, zr.Fall.Zones)
			}
			for j, zone := range test.expectedFallthrough.Zones {
				if zr.Fall.Zones[j] != zone {
					t.Errorf("Test %d, expected fallthrough zone %q, got: %q", i, zone, zr.Fall.Zones[j])
				}
			}
		}
	}
}
