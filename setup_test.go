package zoneregistry

import (
	"net"
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/fall"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input               string
		shouldErr           bool
		expectedZone        string
		expectedZones       int
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
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: nil,
		},
		{
			input: `zoneregistry example.org {
						ttl 100
						interval 20
						timeout 10
					}`,
			shouldErr:           false,
			expectedZone:        "example.org.",
			expectedZones:       1,
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
			expectedTTL:         ttlDefault,
			expectedInterval:    intervalDefault,
			expectedTimeout:     timeoutDefault,
			expectedFallthrough: &fall.F{Zones: []string{"example.com.", "."}},
		},
		// Error tests
		{
			input: `zoneregistry example.org {
						ttl string_not_uint32
					}`,
			shouldErr:           true,
			expectedZone:        "example.org.",
			expectedZones:       1,
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

		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
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
		// Validate TTL
		if !test.shouldErr && zr.TTL != test.expectedTTL {
			t.Errorf("Test %d, expected TTL %d, got: %d", i, test.expectedTTL, zr.TTL)
		}
		// Validate Interval
		if !test.shouldErr && zr.Interval != test.expectedInterval {
			t.Errorf("Test %d, expected INTERVAL %d, got: %d", i, test.expectedInterval, zr.Interval)
		}
		// Validate Timeout
		if !test.shouldErr && zr.Timeout != test.expectedTimeout {
			t.Errorf("Test %d, expected TIMEOUT %d, got: %d", i, test.expectedTimeout, zr.Timeout)
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

func TestParsePeer(t *testing.T) {
	tests := []struct {
		input            string
		shouldErr        bool
		expectedHost     string
		expectedLabels   string
		expectedRole     string
		expectedIPv4     net.IP
		expectedIPv6     net.IP
		expectedProtocol string
		expectedPath     string
		expectedPort     uint32
	}{
		{
			input:            `peer peer1`,
			shouldErr:        false,
			expectedHost:     "peer1.",
			expectedRole:     roleDefault,
			expectedProtocol: protocolDefault,
			expectedPath:     pathDefault,
			expectedPort:     portDefault,
		},
		{
			input: `peer peer1 {
						role primary
						ipv4 10.10.10.10
					}`,
			shouldErr:        false,
			expectedHost:     "peer1.",
			expectedRole:     "primary",
			expectedIPv4:     net.IPv4(10, 10, 10, 10),
			expectedProtocol: protocolDefault,
			expectedPath:     pathDefault,
			expectedPort:     portDefault,
		},
		{
			input: `peer peer1 {
						role asdf
					}`,
			shouldErr: true,
		},
	}
	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		c.Next()
		p, err := parsePeer(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found %s for input %s", i, err, test.input)
		}

		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
		}
		// Validate host
		if !test.shouldErr && p.Host != test.expectedHost {
			t.Errorf("Test %d, expected host %s, got: %s", i, test.expectedHost, p.Host)
		}
		// Validate role
		if !test.shouldErr && p.Role != test.expectedRole {
			t.Errorf("Test %d, expected role %s, got: %s", i, test.expectedRole, p.Role)
		}
		// Validate ipv4
		if !test.shouldErr && !p.IPv4.Equal(test.expectedIPv4) {
			t.Errorf("Test %d, expected ipv4 %s, got: %s", i, test.expectedIPv4.String(), p.IPv4.String())
		}
		// Validate ipv6
		if !test.shouldErr && !p.IPv6.Equal(test.expectedIPv6) {
			t.Errorf("Test %d, expected ipv6 %s, got: %s", i, test.expectedIPv6.String(), p.IPv6.String())
		}
		// Validate protocol
		if !test.shouldErr && p.Protocol != test.expectedProtocol {
			t.Errorf("Test %d, expected protocol %s, got: %s", i, test.expectedProtocol, p.Protocol)
		}
		// Validate path
		if !test.shouldErr && p.Path != test.expectedPath {
			t.Errorf("Test %d, expected path %s, got: %s", i, test.expectedPath, p.Path)
		}
		// Validate port
		if !test.shouldErr && p.Port != test.expectedPort {
			t.Errorf("Test %d, expected port %d, got: %d", i, test.expectedPort, p.Port)
		}
	}
}
