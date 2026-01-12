# wol-web

Web-based Wake-on-LAN management with real-time device monitoring, ARP discovery, and per-host network interface selection.

Custom Go REST API with SQLite database and SvelteKit frontend.

![](https://i.imgur.com/2pGGr1Z.png)

## Quick Start

### Docker Compose (Recommended)

```bash
docker compose up -d
```

Access at `http://localhost:8090`

**View logs:**
```bash
docker logs -f wol-web
```

### Configuration

#### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LISTEN_ADDRESS` | Bind address (e.g., `:8090`, `127.0.0.1:3000`) | `:8090` |
| `URL_PREFIX` | Path for reverse proxy (e.g., `/wol`) | `""` |
| `DEFAULT_NETWORK_INTERFACE` | Global interface(s) - **supports comma-separated** | `""` |
| `ENABLE_PER_HOST_INTERFACES`| Allow per-host interface selection | `false` |
| `BEHIND_PROXY` | Enable secure cookies for HTTPS proxy | `false` |
| `LOG_LEVEL` | Log level (`debug`, `info`, `warning`, `error`) | `info` |
| `LOG_OUTPUT_MODE` | Log output (`stdout`, `file`, `both`) | `stdout` |

#### config.json (auto-created on first run)

```json
{
  "listen_address": ":8090",
  "url_prefix": "",
  "default_network_interface": "",
  "enable_per_host_interfaces": false,
  "ping_timeout_seconds": 5,
  "auth_expire_hours": 4,
  "use_auth": true,
  "readonly_mode": false,
  "behind_proxy": false,
  "health_check_enabled": true,
  "log_level": "info",
  "log_output_mode": "stdout"
}
```

See [CONFIG.md](./CONFIG.md) for all options.

---

## Features

- **Wake-on-LAN:** Send magic packets to wake devices
- **Device Monitoring:** Real-time status (15s intervals) via ARP ping
- **Static IP Support:** Directly ping specific IPs with optional fallback to ARP discovery
- **ARP Discovery:** Scan network and detect devices (Linux only)
- **Network Interfaces:** Per-host or global interface selection with **multiple interface support** for automatic fallback (Linux only)
- **User Management:** Multi-user with superuser roles
- **API:** RESTful endpoints for automation
- **Responsive UI:** Built with SvelteKit and shadcn/ui

---

## Platform Support

### Linux (Full functionality)
- ARP discovery and scanning
- Per-host network interface selection
- Real-time device status monitoring

### Windows/macOS (Limited)
- Basic Wake-on-LAN only
- No ARP discovery/pings
- **Note:** Docker Desktop (Mac/Windows) does not support `--network=host`. Run binary directly for WoL functionality.

---

## Installation (Local System)

### Prerequisites
- Bun
- Go 1.23+

### Steps
```bash
# 1. Clone and install
git clone https://github.com/yourusername/wol-web-extended.git
cd wol-web-extended
bun install

# 2. Build and Run
bun run build
cd apps/server
./wol-server
```

Access at `http://localhost:8090/auth` to create the first superuser.

### Optional: Grant ARP capabilities (Linux)
```bash
sudo setcap cap_net_raw,cap_net_admin+ep apps/server/wol-server
```

---

## Documentation
- [RUNNING.md](./RUNNING.md) - Deployment and configuration
- [CONFIG.md](./CONFIG.md) - Configuration reference
- [LOGGING.md](./LOGGING.md) - Logging guide

---

## License
MIT
