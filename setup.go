package zoneregistry

import (
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
	go zr.Peers.StartHealthChecks()

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
				zr.ttl = uint32(t)

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
				zr.Peers.Interval = uint32(t)

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
				zr.Peers.Timeout = uint32(t)

			case "peers":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return nil, c.ArgErr()
				}
				for _, host := range args {
					p := NewPeer(host)
					if p == nil {
						return nil, c.Errf("Registering peer %s failed", host)
					}
					if err := p.resolveHost(); err != nil {
						return nil, c.Errf("Resolving peer %s failed - %v", host, err)
					}
					zr.Peers.AddPeer(p)
				}

			default:
				return nil, c.Errf("Unknown property '%s'", c.Val())
			}
		}
	}
	return zr, nil
}
