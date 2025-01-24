# zoneregistry

A CoreDNS plugin to handle service discovery across distributed kubernetes clusters

## TODOs

- [x] Return A/AAAA instead of CNAMEs from the registry
- [x] Loadbalance the peers records
- [x] Add concurrency safety on the peers map
- [x] Add e2e test for standalone deployment
- [x] Add primary/secondary role for peers
- [x] Add prometheus metrics integration
- [ ] Add CI + automated testing
- [ ] Add support for multiple zoneregistry masters
- [ ] Add e2e test for k8s deployment
- [ ] Update README

## Configure

Configuration options can be used to customize the behaviour of a plugin:

```
{
zoneregistry ZONE
    peers [PEERS...]
    interval INTERVAL
    ttl TTL
    fallthrough [ZONES...]
}
```

- `peers` the subzones to run healthchecks against.
- `interval` can be used to override the default INTERVAL value of 60 seconds.
- `ttl` can be used to override the default TTL value of 300 seconds.
- `fallthrough` if zone matches and no record can be generated, pass request to the next plugin. If **[ZONES...]** is omitted, then fallthrough happens for all zones for which the plugin is authoritative. If specific zones are listed (for example `in-addr.arpa` and `ip6.arpa`), then only queries for those zones will be subject to fallthrough.

## Example

Configuring the zone registry to perform healthchecks on 3 k8s clusters

```
. {
  zoneregistry service.pinax.network {
    debug
    metrics
    interval 60
    timeout 10
    ttl 300

    peer riv-prod1.service.pinax.network {
        role primary
        labels cluster-env=prod
        IPv4 172.100.0.101
        IPv6 2001:db8:172:100::101
        protocol http
        path /health
        port 8080
    }
    peer riv-prod2.service.pinax.network {
        role primary
        labels cluster-env=prod
        IPv4 172.100.0.102
        IPv6 2001:db8:172:100::102
        protocol http
        path /health
        port 8080
    }
    peer mar-dev1.service.pinax.network {
        role secondary
        labels cluster-env=dev
        IPv4 172.100.0.1043
        IPv6 2001:db8:172:100::103
        protocol http
        path /health
        port 8080
    }
  }
}
```
