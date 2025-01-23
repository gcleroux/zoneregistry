package zoneregistry

import (
	"net"
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

const pluginName = "zoneregistry"

var log = clog.NewWithPlugin(pluginName)

// init registers this plugin.
func init() { plugin.Register(pluginName, setup) }

// setup is the function that gets called when the config parser see the token "example". Setup is responsible
// for parsing any extra options the example plugin may have. The first token this function sees is "example".
func setup(c *caddy.Controller) error {
	zr, err := parse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}
	go zr.StartHealthChecks()

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		zr.Next = next
		return zr
	})

	// All OK, return a nil error.
	return nil
}

func parse(c *caddy.Controller) (*ZoneRegistry, error) {
	zr := newZoneRegistry()

	for c.Next() {
		zones := c.RemainingArgs()
		zr.Zones = zones

		if len(zr.Zones) == 0 {
			zr.Zones = make([]string, len(c.ServerBlockKeys))
			copy(zr.Zones, c.ServerBlockKeys)
		}

		for i, str := range zr.Zones {
			if host := plugin.Host(str).NormalizeExact(); len(host) != 0 {
				zr.Zones[i] = host[0]
			}
		}

		for c.NextBlock() {
			switch c.Val() {

			case "fallthrough":
				zr.Fall.SetZonesFromArgs(c.RemainingArgs())

			case "ttl":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return nil, c.ArgErr()
				}
				t, err := strconv.Atoi(args[0])
				if err != nil {
					return nil, err
				}
				if t < 0 || t > 3600 {
					return nil, c.Errf("ttl must be in range [0, 3600]: %d", t)
				}
				zr.TTL = uint32(t)

			case "interval":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return nil, c.ArgErr()
				}
				t, err := strconv.Atoi(args[0])
				if err != nil {
					return nil, err
				}
				if t < 0 || t > 300 {
					return nil, c.Errf("interval must be in range [0, 300]: %d", t)
				}
				zr.Interval = uint32(t)

			case "timeout":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return nil, c.ArgErr()
				}
				t, err := strconv.Atoi(args[0])
				if err != nil {
					return nil, err
				}
				if t < 0 || t > 30 {
					return nil, c.Errf("timeout must be in range [0, 30]: %d", t)
				}
				zr.Timeout = uint32(t)

			case "peer":
				peer, err := parsePeer(c)
				if err != nil {
					return nil, err
				}
				zr.mu.Lock()
				zr.Peers = append(zr.Peers, peer)
				zr.mu.Unlock()
			default:
				return nil, c.Errf("Unknown property '%s'", c.Val())
			}
		}
	}
	return zr, nil
}

func parsePeer(c *caddy.Controller) (*Peer, error) {
	peer := NewPeer()

	args := c.RemainingArgs()
	if len(args) == 0 {
		return nil, c.ArgErr()
	}
	if h := plugin.Host(args[0]).NormalizeExact(); len(h) != 0 {
		peer.Host = h[0]
	}

	for c.Next() {
		switch c.Val() {

		case "role":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			if args[0] != "primary" && args[0] != "secondary" {
				return nil, c.Errf("role must be ['primary', 'secondary']: %s", args[0])
			}
			peer.Role = args[0]

		case "labels":
			peer.Labels = c.RemainingArgs()

		case "ipv4":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			ip := net.ParseIP(args[0])
			if ip == nil || ip.To4() == nil {
				return nil, c.Errf("invalid IPv4: %s", args[0])
			}
			peer.IPv4 = ip

		case "ipv6":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			ip := net.ParseIP(args[0])
			if ip == nil || ip.To4() != nil {
				return nil, c.Errf("invalid IPv6: %s", args[0])
			}
			peer.IPv6 = ip

		case "protocol":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			if args[0] != "http" && args[0] != "https" {
				return nil, c.Errf("protocol must be ['http', 'https']: %s", args[0])
			}
			peer.Protocol = args[0]

		case "path":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			peer.Path = args[0]

		case "port":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			p, err := strconv.Atoi(args[0])
			if err != nil {
				return nil, err
			}
			if p < 0 || p > 65535 {
				return nil, c.Errf("port must be in range [0, 65535]: %d", p)
			}
			peer.Port = uint32(p)

		// Must manually check for blocks since c.NextBlock doesn't support nesting
		case "{":
			// Opening the peer block
			continue
		case "}":
			// Closing the peer block
			return peer, nil

		default:
			return nil, c.Errf("Unknown property '%s'", c.Val())
		}
	}
	return peer, nil
}
