package zoneregistry

import (
	"testing"
)

func TestPeers(t *testing.T) {
	tests := []struct {
		input            string
		expectedPeerHost string
	}{
		{
			input:            `example.org`,
			expectedPeerHost: "example.org",
		},
	}

	for i, test := range tests {
		peer := NewPeer()
		peer.Host = test.input

		// Validate Peer Host
		if test.expectedPeerHost != peer.Host {
			t.Errorf("Test %d, expected %s host for input %s, got: %s", i, test.expectedPeerHost, test.input, peer.Host)
		}

	}
}
