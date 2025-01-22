# zoneregistry

A CoreDNS plugin to handle service discovery across distributed kubernetes clusters

## TODOs

- [x] Return A/AAAA instead of CNAMEs from the registry
- [x] Loadbalance the peers records
- [x] Add concurrency safety on the peers map
- [ ] Add CI + automated testing
- [ ] Add support for multiple zoneregistry masters
- [ ] Add e2e test for standalone deployment
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
    peers riv-prod1. riv-prod2. mar-prod3.
    interval 60
    ttl 300
  }
}

riv-prod1. {
  forward . 172.21.0.1:53
}

riv-prod2. {
  forward . 172.21.0.2:53
}

mar-prod3. {
  forward . 172.19.0.1:53
}
```
