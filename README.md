# psstd

[![CI](https://github.com/OffPeakEngineer/psstd/actions/workflows/ci.yml/badge.svg)](https://github.com/OffPeakEngineer/psstd/actions/workflows/ci.yml) [![Release](https://github.com/OffPeakEngineer/psstd/actions/workflows/release.yml/badge.svg)](https://github.com/OffPeakEngineer/psstd/actions/workflows/release.yml)

**psstd is a resilient cluster htop.** Run it on a few machines, open any node in a browser, and watch the whole cluster from a server-rendered dashboard. If the node serving your browser gets busy, it can send the next refresh to a quieter peer.

The result is intentionally simple: every node can serve the UI, every node shares fresh load metrics with its peers, and the browser can be hot-potatoed around the cluster without a central coordinator.

## Features

- **Cluster htop view**: CPU, memory, load, freshness, and offline status for every known node.
- **Hot-potato refresh**: a busy node can bake a lower-load peer into the next browser refresh.
- **Peer rebasing**: node links point at that node's own HTTP address, so the browser moves to the selected peer.
- **Zero-config LAN discovery**: nodes discover peers with mDNS.
- **Seed support**: provide explicit peers with `PSSTD_SEEDS` when discovery is not enough.
- **Local-first resilience**: each node keeps enough recent state to keep rendering during peer churn.

## Quick Start

```bash
go install github.com/OffPeakEngineer/psstd@latest
psstd
```

Or build from a checkout:

```bash
go build ./
./psstd
```

Open:

```text
http://localhost:8080
```

Run the same binary on additional LAN machines and they should discover each other automatically. If the advertised browser URL needs to differ from the listen address, set `PSSTD_ADVERTISE_HTTP`.

If you start `psstd` again while an instance is already running on the machine, the second process does not open another writer on the same store or publish a duplicate node. When the existing `PSSTD_DB` is locked, or the configured gossip port is already bound, it joins as a terminal-only mirror with a temporary database and renders the cluster view in your terminal.

## Configuration

```bash
export PSSTD_HTTP=":9000"                         # HTTP listen address, default :8080
export PSSTD_ADVERTISE_HTTP="http://10.0.1.25:9000" # browser-reachable URL for this node
export PSSTD_GOSSIP=":7947"                       # peer sync listen address, default :7946
export PSSTD_SEEDS="10.0.1.20:7946,10.0.1.21:7946" # explicit peer sync addresses
export PSSTD_DB="./data"                          # local state directory
export PSSTD_WEB="true"                           # set false for sync-only nodes
export PSSTD_NODE_NAME="rack-a-01"                # optional stable node identity override
export PSSTD_NODE_TTL="15s"                       # how long this node's heartbeat stays online
./psstd
```

By default, psstd uses the OS hostname as the node identity. Set
`PSSTD_NODE_NAME` for cloned hosts, containers, or multiple test instances that
would otherwise publish the same hostname. The override must be non-empty and
must not contain whitespace.

Each node publishes its own heartbeat TTL with `PSSTD_NODE_TTL`. Shorter values
make stale/offline indication react faster; longer values are better for slow or
lossy networks. The default is `15s`, and values must be at least `2s`.

## Discovery

| Environment | How nodes find each other |
|---|---|
| LAN / bare metal | mDNS service `_psstd._tcp` |
| Static hosts | `PSSTD_SEEDS` |
| Mixed setup | mDNS discoveries and explicit seeds are merged |
| Single node | Renders solo until peers appear |

## Service Templates

Templates live in `deploy/` for common ways to keep psstd running:

| Target | Template |
|---|---|
| Linux systemd | `deploy/systemd/psstd.service` and `deploy/systemd/psstd.env` |
| macOS launchd | `deploy/launchd/com.offpeakengineer.psstd.plist` |
| Windows service | `deploy/windows/install-service.ps1` using NSSM |
| Kubernetes | `deploy/kubernetes/psstd.yaml` |
| Ansible local binary | `deploy/ansible/install-psstd.yml` |
| Ansible release binary | `deploy/ansible/install-release-psstd.yml` |
| Traefik to bare metal | `deploy/traefik/bare-metal-node.yaml` |
| Traefik single hostname | `deploy/traefik/single-host-query.yaml` |

Linux systemd quick install from a built binary:

```bash
go build -o psstd ./
sudo sh deploy/systemd/install.sh
sudoedit /etc/psstd/psstd.env
sudo systemctl restart psstd
```

For clusters, build or download the binary once, then distribute that binary with systemd or Ansible. Avoid `go install` on every host unless you intentionally manage Go toolchains there.

macOS launchd:

```bash
sudo install -m 0755 psstd /usr/local/bin/psstd
sudo mkdir -p /usr/local/var/psstd /usr/local/var/log
sudo cp deploy/launchd/com.offpeakengineer.psstd.plist /Library/LaunchDaemons/
sudo launchctl bootstrap system /Library/LaunchDaemons/com.offpeakengineer.psstd.plist
```

Windows service:

```powershell
.\deploy\windows\install-service.ps1 -BinaryPath "C:\Program Files\psstd\psstd.exe" -AdvertiseHttp "http://10.0.1.25:8080"
```

Kubernetes:

```bash
kubectl apply -f deploy/kubernetes/psstd.yaml
```

### Reverse Proxies

Hot-potato refreshes and peer links require node identity to survive the browser round trip. If a single URL like `https://psstd.example.com` is backed by a normal load balancer, the next request may land on any node, so the browser has not really rebased to the lower-load peer.

Use one browser-routable URL per node instead:

```text
https://psstd-node-a.example.com -> 10.0.1.25:8080
https://psstd-node-b.example.com -> 10.0.1.26:8080
https://psstd-node-c.example.com -> 10.0.1.27:8080
```

Then set each node's advertised URL to its proxied hostname:

```bash
PSSTD_HTTP=:8080
PSSTD_ADVERTISE_HTTP=https://psstd-node-a.example.com
PSSTD_GOSSIP=:7946
PSSTD_SEEDS=10.0.1.26:7946,10.0.1.27:7946
```

With Traefik in Kubernetes proxying bare-metal nodes, create one Service, Endpoints, and IngressRoute per node. Start from `deploy/traefik/bare-metal-node.yaml`.

If you prefer one hostname, route by query parameter instead:

```text
https://psstd.example.com/?psstd_node=node-a -> 10.0.1.25:8080
https://psstd.example.com/?psstd_node=node-b -> 10.0.1.26:8080
```

Then advertise the routed URL from each node:

```bash
PSSTD_ADVERTISE_HTTP=https://psstd.example.com/?psstd_node=node-a
```

Traefik supports query-param matchers in router rules, so this keeps link clicks and hot-potato refreshes on a single DNS name while still selecting a specific backend. Start from `deploy/traefik/single-host-query.yaml`.

That example also includes a low-priority host-only fallback route. If Traefik sees no matching `psstd_node` value, it sends the request to a shared `psstd-any` service instead of pinning fallback traffic to one node. Keep the `psstd-any` Endpoints list limited to nodes that should receive unrouted fallback traffic.

## Notes

psstd uses peer-to-peer state sharing and a small local store internally, but those are implementation details for the dashboard. It is not intended to be a general-purpose distributed database or key-value API.

## Requirements

- Go 1.25+ if building from source with `go install` or `go build`.

For fleet installs, prefer release binaries or a binary built once in CI over compiling on every host. This avoids distro Go version drift and keeps bare-metal installs simple.

CI runs on GitHub-hosted Linux, Windows, and macOS runners for both x64 and arm64 where standard hosted runners are available. Releases publish matching Linux, macOS, and Windows binaries from `deploy/release/build-all.sh`.
