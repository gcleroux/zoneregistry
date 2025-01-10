package zoneregistry

import (
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
		// 	zones := c.RemainingArgs()
		// 	gw.Zones = zones
		//
		// 	if len(gw.Zones) == 0 {
		// 		gw.Zones = make([]string, len(c.ServerBlockKeys))
		// 		copy(gw.Zones, c.ServerBlockKeys)
		// 	}
		//
		// 	for i, str := range gw.Zones {
		// 		if host := plugin.Host(str).NormalizeExact(); len(host) != 0 {
		// 			gw.Zones[i] = host[0]
		// 		}
		// 	}
		//
		// 	for c.NextBlock() {
		// 		switch c.Val() {
		// 		case "fallthrough":
		// 			gw.Fall.SetZonesFromArgs(c.RemainingArgs())
		// 		case "secondary":
		// 			args := c.RemainingArgs()
		// 			if len(args) == 0 {
		// 				return nil, c.ArgErr()
		// 			}
		// 			gw.secondNS = args[0]
		// 		case "resources":
		// 			args := c.RemainingArgs()
		//
		// 			gw.updateResources(args)
		//
		// 			if len(args) == 0 {
		// 				return nil, c.Errf("Incorrectly formated 'resource' parameter")
		// 			}
		// 		case "ttl":
		// 			args := c.RemainingArgs()
		// 			if len(args) == 0 {
		// 				return nil, c.ArgErr()
		// 			}
		// 			t, err := strconv.Atoi(args[0])
		// 			if err != nil {
		// 				return nil, err
		// 			}
		// 			if t < 0 || t > 3600 {
		// 				return nil, c.Errf("ttl must be in range [0, 3600]: %d", t)
		// 			}
		// 			gw.ttlLow = uint32(t)
		// 		case "apex":
		// 			args := c.RemainingArgs()
		// 			if len(args) == 0 {
		// 				return nil, c.ArgErr()
		// 			}
		// 			gw.apex = args[0]
		// 		case "kubeconfig":
		// 			args := c.RemainingArgs()
		// 			if len(args) == 0 {
		// 				return nil, c.ArgErr()
		// 			}
		// 			gw.configFile = args[0]
		// 			if len(args) == 2 {
		// 				gw.configContext = args[1]
		// 			}
		// 		default:
		// 			return nil, c.Errf("Unknown property '%s'", c.Val())
		// 		}
		// 	}
	}
	return zr, nil
}
