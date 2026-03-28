# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ThetisTunnel is a **distribution-only repository** — it contains pre-compiled C++ binaries, documentation, and a license. There is no source code. The tool tunnels UDP traffic between the [Thetis SDR application](https://github.com/TAPR/OpenHPSDR-Thetis) and a remote radio over the internet using a single TCP connection, enabling remote radio operation without a VPN.

## Repository Contents

- Pre-compiled binaries for 4 platforms (Windows x64, Linux x86_64, Linux aarch64, Linux armhf)
- `README.md` — command-line reference and usage examples
- `ThetisTunnelC.png` / `ThetisTunnelC.pdf` — network architecture diagram
- `LICENCE` — proprietary free-use license (Blitter8 Ltd); no modification, reverse engineering, or redistribution

## Architecture

Two instances run simultaneously — one at each end of the tunnel:

```
[Thetis app]  ←UDP→  [ThetisTunnelC --mode=connect]  ←TCP→  [ThetisTunnelC --mode=listen]  ←UDP→  [Radio]
  Thetis site         intercepts UDP ports 1024-1029          radio site (port-forwarded)           ports 1024-1042
```

**Listen mode** (radio site): Binds a TCP listener, accepts one client, and proxies UDP to/from the radio via a single local proxy port. Optionally broadcasts port 1024 discovery frames on the radio subnet.

**Connect mode** (Thetis site): Connects outbound to the listener, intercepts Thetis UDP traffic, and tunnels it through the TCP connection. Supports DNS hostnames for the listener address.

**Shared features**: Optional shared-secret authentication (`--key`), real-time throughput display with configurable rate scaling (`--maxInRate`/`--maxOutRate`), graceful shutdown via `Q` or `Ctrl+C`, peak rate reset via `R`.

## No Build or Test Process

There are no build files, tests, or CI/CD pipelines — work in this repository is limited to updating documentation, the architecture diagram, and distributing new binary releases.
