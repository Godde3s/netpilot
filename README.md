# netpilot 🛩️

**A single-binary network toolkit in pure Go** — TCP port scanner, latency probe, HTTP health checks, DNS inspector and IPv4 CIDR calculator. Zero dependencies, colored TTY output, ~5 MB binary.

[![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Author](https://img.shields.io/badge/author-@Godde3s-181717?logo=github)](https://github.com/Godde3s)

> Built as a real-world exercise in clean Go: small focused packages, concurrency with worker pools, graceful ANSI degradation on non-TTY stdout, and honest exit codes (0 healthy · 2 unhealthy) so it drops straight into cron jobs and CI pipelines.

## Install

```bash
go install github.com/Godde3s/netpilot@latest
# or build from source
git clone https://github.com/Godde3s/netpilot && cd netpilot
go build -o netpilot .
```

Cross-compile in one line:

```bash
GOOS=linux GOARCH=arm64 go build -o netpilot-linux-arm64 .
```

## Commands

### `ports` — concurrent TCP scan

```bash
netpilot ports example.com -p 22,80,443
netpilot ports 10.0.0.1 -p 1-1000 -w 512 -v   # 512 workers, show closed too
```

```
 ● scanning example.com — 3 ports · 256 workers
 ✓ port 80     open   41ms
 ✓ port 443    open   43ms
 ● done in 44ms — 2 open
```

### `ping` — latency without ICMP privileges

```bash
netpilot ping example.com -P 443 -c 10
```

```
 ✓ seq=1 from example.com:443 time=12.4ms
 ✓ seq=2 from example.com:443 time=11.9ms
 ● target summary — healthy
   sent 10 · recv 10 · loss 0.0%
   min 11.6ms · avg 12.3ms · max 13.8ms · jitter 640µs
```

### `health` — uptime checks (CI-friendly exit codes)

```bash
netpilot health https://example.com https://api.example.com/healthz
```

### `dns` — record inspection via Go's native resolver

```bash
netpilot dns github.com -t MX
netpilot dns example.com -t TXT
```

### `subnet` — IPv4 CIDR math

```bash
netpilot subnet 10.0.4.0/22 -n 5
```

```
 ● CIDR 10.0.4.0/22
   network    10.0.4.0
   broadcast  10.0.7.255
   first host 10.0.4.1
   last host  10.0.7.254
   netmask    255.255.252.0
   addresses  1,024 total · 1,022 usable
```

## Why

Most quick network checks end up stitching together `nc`, `curl`, `dig` and spreadsheet math. `netpilot` puts the five commands I actually use into one dependency-free binary that compiles anywhere Go does — servers, routers-on-steroids, Raspberry Pis, CI runners.

## Ethics

Scan only systems you own or are authorized to test. The tool is deliberately polite (connect-only, no SYN stealth, bounded workers) — use it as an engineer, not an attacker.

---

Part of [my open-source portfolio](https://godde3s.github.io/Usf/) — built with agentic coding (Claude Code + Hermes Agent). MIT licensed.
