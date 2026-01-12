# Running WoL-Web

## Container Deployment (Recommended)

### Docker Compose

```bash
docker compose up -d
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LISTEN_ADDRESS` | Bind address (default `:8090`) |
| `URL_PREFIX` | Reverse proxy path (e.g., `/wol`) |
| `DEFAULT_NETWORK_INTERFACE` | Global interface(s) (comma-separated) |
| `ENABLE_PER_HOST_INTERFACES`| Allow per-host interface selection |
| `BEHIND_PROXY` | Set to `true` for HTTPS reverse proxies |
| `LOG_LEVEL` | `debug`, `info`, `warning`, `error` |
| `LOG_OUTPUT_MODE` | `stdout`, `file`, `both` |

### Manual Docker Run

```bash
docker run -d \
  --name wol-web \
  --network=host \
  --cap-add=NET_RAW \
  --cap-add=NET_ADMIN \
  -v wol_data:/app/data \
  wol-web:local
```

---

## Binary Deployment

### Build

```bash
bun run build
```

### Run

```bash
cd apps/server

# Custom config and database
./wol-server -config ./config.json -db ./wol.db

# Enable debug logging (CLI override)
./wol-server -debug

# Reset superuser password (interactive)
./wol-server --reset-admin
```

### Linux Capabilities (ARP)

For ARP ping and cache flushing on Linux, grant capabilities:

```bash
sudo setcap cap_net_raw,cap_net_admin+ep /path/to/wol-server
```

---

## Usage Scenarios

**1. Simple setup (default)**
Authentication enabled, uses all available network interfaces.

**2. Specific interface for all hosts**
`DEFAULT_NETWORK_INTERFACE=eth0` - All WoL/pings go through eth0.

**3. Multi-subnet failover**
`DEFAULT_NETWORK_INTERFACE=eth0,eth1` - Tries eth0 first, then eth1 for pings; WoL broadcasts to both.

**4. Static IP with Fallback**
Configure a host with `Static IP` and `Use as Fallback = true`. It will try to find the device via MAC address first, then try the Static IP if resolution fails.

---

## First Time Setup

1. Start the server.
2. Navigate to `http://localhost:8090/auth`.
3. Create the first superuser via the setup form.

---

## Troubleshooting

### WoL not working
- Enable WoL in target device BIOS.
- Check `DEFAULT_NETWORK_INTERFACE` (Linux).
- Ensure `CAP_NET_RAW` is granted.
- Check broadcast address format (e.g., `192.168.1.255:9`).

### Database locked
This application uses SQLite in WAL mode with busy timeouts to prevent locking. If issues persist, ensure the process has write permissions to the database file and directory.

### Forgot password
Use the password reset tool:
```bash
./wol-server --reset-admin
```
