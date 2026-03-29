# ThetisTunnel

A tool that tunnels the UDP data in/out of Thetis and a Radio, enabling remote radio operation over the internet without a VPN.

## Source Code and Licensing

**The source code for ThetisTunnel is not available and will not be provided.** The software is developed and owned by Richie (MW0LGE) of Blitter8 Ltd and is distributed as compiled binaries only.

The [LICENCE](LICENCE) (Blitter8 Ltd Free Use Software Licence) explicitly prohibits:

- Decompiling, reverse engineering, or disassembling the software
- Attempting to derive the source code by any means
- Modifying, adapting, or creating derivative works
- Redistributing the software without the licensor's express written permission

### Go Re-implementation

I (W5TSU) accept this. I've never used ThetisTunnel but I like the design. So I decided to use Claude to reproduce its function in Go. This is the result.

The Go source lives in the root of this repository. Pre-built binaries for Linux, macOS, and Windows are in the `dist/` directory. This code is release with a GPL2 license.

---

## Building from Source

### Dependencies

| Dependency | Version | Notes |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.21 or later | The only required tool |
| `golang.org/x/term` | v0.21.0 | Fetched automatically by `go mod tidy` |
| `golang.org/x/sys` | v0.21.0 | Indirect dependency, fetched automatically |

No C compiler or CGO is required. All builds use `CGO_ENABLED=0`.

### Build for your current platform

```bash
git clone https://github.com/W5TSU/ThetisTunnel.git
cd ThetisTunnel
go mod tidy
go build -o thetistunnel .
```

### Build all platforms

The `build.sh` script cross-compiles for all supported targets and writes binaries to `dist/`:

```bash
bash build.sh
```

| Output file | Platform |
|---|---|
| `thetistunnel-linux-amd64` | Linux x86-64 |
| `thetistunnel-linux-arm64` | Linux ARM64 (e.g. Raspberry Pi 4/5) |
| `thetistunnel-linux-armhf` | Linux ARMv7 32-bit (e.g. Raspberry Pi 3) |
| `thetistunnel-darwin-amd64` | macOS Intel |
| `thetistunnel-darwin-arm64` | macOS Apple Silicon |
| `thetistunnel-windows-amd64.exe` | Windows x64 |

### Run tests

```bash
go test ./...
```

---

## Usage

Run one instance at the radio site (`--mode=listen`) and one at the Thetis site (`--mode=connect`).

At the radio site you will need to port-forward the TCP port defined by `--tcpPort` on your router to the machine running the listen instance.

```
Usage:
  ThetisTunnelC --mode=listen  --tcpPort=PORT --tcpBindIP=IP --udpBindIP=IP --radioIp=IP --radioProxyPort=PORT [options]
  ThetisTunnelC --mode=connect --tcpHost=IP  --tcpPort=PORT --udpBindIP=IP [--thetisPorts=RANGE] [options]
```

### Required arguments

| Flag | Description |
|---|---|
| `--mode=listen\|connect` | `listen` = radio site; `connect` = Thetis site |
| `--tcpPort=PORT` | TCP port for the tunnel |

### Listen mode (radio site)

| Flag | Description |
|---|---|
| `--tcpBindIP=IP` | Local IP to bind the TCP listener (use `0.0.0.0` for all interfaces) |
| `--udpBindIP=IP` | Local IP to bind UDP sockets (interface that reaches the radio) |
| `--radioIp=IP` | Radio IPv4 address |
| `--radioProxyPort=PORT` | Local UDP port used to send to the radio and receive replies |
| `--radioSrcRange=RANGE` | Radio source UDP port range accepted from the radio (default `1024-1042`) |
| `--broadcast1024` | Broadcast port 1024 discovery frames to the radio subnet and `255.255.255.255` |

### Connect mode (Thetis site)

| Flag | Description |
|---|---|
| `--tcpHost=IP` | Listener public IP, private IP, or hostname (e.g. `radio.mydomain.com`) |
| `--udpBindIP=IP` | Local IP to bind UDP sockets (interface used by Thetis) |
| `--thetisPorts=RANGE` | UDP destination ports to intercept from Thetis (default `1024-1029`) |

### Common options

| Flag | Description |
|---|---|
| `--key=VALUE` | Shared secret key (up to 32 chars); connect must match listen |
| `--noUI` | Disable the live throughput display |
| `--debug` | Log connection and protocol details (auth, per-frame port/length/address) |
| `--maxInRate=VALUE` | Scale IN rate bars to VALUE bits/sec (e.g. `10M`, `1G`) |
| `--maxOutRate=VALUE` | Scale OUT rate bars to VALUE bits/sec |

Port range format: `1024-1042` or `1024,1025,1030-1042`

### Examples

**Radio site** — bind to `192.168.0.76`, port 5000, talk to radio `192.168.0.157` via local proxy port 50005, key `abc123`:
```
ThetisTunnelC --mode=listen --tcpPort=5000 --tcpBindIP=192.168.0.76 --udpBindIP=192.168.0.76 --radioIp=192.168.0.157 --radioProxyPort=50005 --key=abc123
```

**Thetis site** — connect to listener at `82.83.84.85:5000`, bind local UDP to `192.168.0.76`, key `abc123`:
```
ThetisTunnelC --mode=connect --tcpHost=82.83.84.85 --tcpPort=5000 --udpBindIP=192.168.0.76 --key=abc123
```

### Controls

| Key | Action |
|---|---|
| `Q` | Quit |
| `R` | Reset peak rate display |
| `Ctrl+C` | Quit |
