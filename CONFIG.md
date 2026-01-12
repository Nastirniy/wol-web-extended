# Configuration Guide

This guide explains all configuration options for the Wake-on-LAN Web Server.

## Quick Start

The server uses `config.json` for configuration. If the file doesn't exist, a default one will be created automatically.

**Example config.json:**

```json
{
  "listen_address": "127.0.0.1:8090",
  "url_prefix": "/wolweb",
  "default_network_interface": "",
  "enable_per_host_interfaces": true,
  "ping_timeout_seconds": 5,
  "auth_expire_hours": 4,
  "use_auth": true,
  "readonly_mode": false,
  "behind_proxy": true,
  "health_check_enabled": true,
  "log_level": "info",
  "log_output_mode": "stdout",
  "log_dir": "./logs",
  "log_max_size_mb": 100,
  "log_max_age_days": 30,
  "log_rotation": true
}
```

See [config.example.json](config.example.json) for a fully documented example.

## Configuration Options

### listen_address (string)

Server bind address and port in format `address:port`.

**Examples:**

- `:8090` - All network interfaces, port 8090 (default)
- `127.0.0.1:50002` - Localhost only, port 50002
- `0.0.0.0:8080` - All interfaces explicitly, port 8080
- `192.168.1.100:9000` - Specific IP address, port 9000

**Environment Variable:** `LISTEN_ADDRESS`

**Backward Compatibility:** Old `server_port` and `server_address` fields are deprecated but still supported. They will be automatically converted to `listen_address`.

---

### url_prefix (string)

URL prefix for reverse proxy deployments.

**Examples:**

- `""` - Root path (no prefix)
- `"/wolweb"` - Application accessible at http://example.com/wolweb
- `"/wol"` - Application accessible at http://example.com/wol

**Important:**

- Must start with `/` if set
- Used for Nginx, Caddy, Traefik reverse proxy setups
- Leave empty if running standalone

**Environment Variable:** `URL_PREFIX`

---

### default_network_interface (string)

Default network interface(s) used when per-host interfaces are disabled.

**Supports multiple interfaces** as comma-separated values!

**Examples:**

- `""` - Use all available interfaces (default)
- `"eth0"` - Use only eth0 interface
- `"eth0,eth1"` - Use multiple interfaces (tries eth0, then eth1)
- `"eth0,eth1,wlan0"` - Use three interfaces (tries each in order)
- `"wlan0"` - Use wireless interface

**Platform:** Linux only (ignored on Windows/macOS)

**Environment Variable:** `DEFAULT_NETWORK_INTERFACE`

**Multiple Interface Behavior:**

- **Ping operations**: Tries each interface in order until one succeeds
  - **IMPORTANT**: Only use with non-overlapping IP ranges across network segments
  - **WARNING**: If multiple segments have the same IP range, ping may detect wrong device
  - Example problem: eth0 (192.168.1.0/24) and eth1 (192.168.1.0/24) both have device at 192.168.1.100
  - Not recommended for complex network topologies (GRE tunnels, overlapping subnets, etc.)
- **WoL operations**: Broadcasts packet to ALL specified interfaces simultaneously
- Provides automatic fallback if primary interface fails (ping)
- Ensures maximum reach for WoL packets across all network segments
- Useful for multi-homed servers with non-overlapping network ranges or redundant paths to same network

**Usage Notes:**

- Only used when `enable_per_host_interfaces` is false
- When `enable_per_host_interfaces` is true, hosts without interface use all interfaces
- See "Network Interface Modes" section below for detailed examples

---

### enable_per_host_interfaces (boolean)

Allow hosts to specify their own network interfaces.

**Values:**

- `true` - Each host can specify its own network interface(s)
- `false` - All hosts use `default_network_interface` setting (default)

**Platform:** Linux only

**Behavior:**

- When `false`: All hosts use `default_network_interface` (or all interfaces if not specified)
- When `true`: Hosts can specify own interface(s), otherwise use all interfaces
- **Per-host interfaces also support multiple interfaces** (comma-separated, e.g., "eth0,eth1")
- Works in both auth and no-auth modes
- Disabled in `readonly_mode`

**Environment Variable:** `ENABLE_PER_HOST_INTERFACES` (set to `true` or `1`)

**Multiple Interfaces Per Host:**
When enabled, each host can specify multiple interfaces like `"eth0,eth1"` for automatic fallback and redundancy.

---

### ping_timeout_seconds (integer)

Timeout for ping/status check operations.

**Range:** 1-60 seconds

**Default:** 5 seconds

**Environment Variable:** `PING_TIMEOUT_SECONDS`

**Notes:**

- Used for ping/status check operations
- Shorter timeout = faster checks
- Longer timeout = more reliable

---

### auth_expire_hours (integer)

Session expiration time in hours.

**Range:** Minimum 1 hour

**Default:** 4 hours

**Environment Variable:** `AUTH_EXPIRE_HOURS`

**Notes:**

- Users must re-login after this period
- Only applies when `use_auth: true`
- Sessions are checked every 10 minutes for cleanup

---

### use_auth (boolean)

Enable user authentication system.

**Values:**

- `true` - Authentication required (default)
- `false` - Public access, no login needed

**Environment Variable:** `USE_AUTH` (set to `true` or `1`)

**When true:**

- Users must login to access the system
- Each user has their own hosts
- Superuser can manage other users
- First-time setup creates superuser via web UI or API

**When false:**

- No login required
- All features publicly accessible
- All hosts visible to everyone
- Not recommended for internet-facing deployments

---

### readonly_mode (boolean)

Disable host creation, modification, and deletion.

**Values:**

- `true` - Read-only mode (WoL packets still allowed)
- `false` - Full access (default)

**Environment Variable:** `READONLY_MODE` (set to `true` or `1`)

**When enabled:**

- Cannot create new hosts
- Cannot edit existing hosts
- Cannot delete hosts
- Can still send WoL packets
- Can still view host status

**Use cases:**

- Kiosk mode deployments
- Shared viewing access
- Prevent accidental modifications

---

### behind_proxy (boolean)

Indicates if running behind HTTPS reverse proxy.

**Values:**

- `true` - Behind HTTPS proxy (enables secure cookies)
- `false` - Direct access (default)

**Environment Variable:** `BEHIND_PROXY` (set to `true` or `1`)

**When true:**

- Session cookies marked as Secure
- Requires HTTPS reverse proxy (Nginx, Caddy, Traefik, etc.)

**When false:**

- Cookies not marked as Secure
- Works with HTTP
- **WARNING:** Credentials sent in plaintext over network

**Security Note:** Always use `true` in production with HTTPS!

---

### debug (boolean)

Enable detailed debug logging.

**Values:**

- `true` - Debug mode enabled
- `false` - Normal logging (default)

**Environment Variable:** `DEBUG` (set to `true` or `1`)

**Command-line Flag:** `-debug` or `--debug`

**When enabled, logs include:**

- WoL packet sending attempts
- Network interface discovery and selection
- ARP table lookups
- Detailed error messages
- Configuration loading details

**Example debug output:**

```
[DEBUG] SendWakeOnLan called: MAC=aa:bb:cc:dd:ee:ff, Target=192.168.1.255:9, Interfaces=(all)
[DEBUG] Found 3 network interfaces
[DEBUG] Attempting to send from interface eth0 (IP: 192.168.1.100)
SUCCESS: WoL packet sent via interface eth0 (IP: 192.168.1.100) to 192.168.1.255:9
```

**Use cases:**

- Troubleshooting WoL packet issues
- Debugging network interface problems
- Understanding configuration loading
- Service debugging via journalctl

---

### health_check_enabled (boolean)

Enable the health check endpoint for monitoring and load balancer health checks.

**Values:**

- `true` - Health check endpoint enabled (default)
- `false` - Health check endpoint disabled (returns 404)

**Environment Variable:** `HEALTH_CHECK_ENABLED` (set to `true` or `1`)

**Endpoint:** `GET /api/health`

**Response format:**

```json
{
  "status": "ok",
  "timestamp": "2025-11-10T12:34:56Z",
  "version": "1.0.0"
}
```

**Features:**

- No authentication required
- Returns HTTP 200 with JSON status when enabled
- Returns HTTP 404 when disabled
- Useful for Docker healthchecks, Kubernetes probes, and load balancers

**Example usage with Docker:**

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8090/api/health"]
  interval: 30s
  timeout: 5s
  retries: 3
```

**Example usage with Kubernetes:**

```yaml
livenessProbe:
  httpGet:
    path: /api/health
    port: 8090
  initialDelaySeconds: 10
  periodSeconds: 30
```

**Use cases:**

- Container orchestration health checks
- Load balancer health monitoring
- Uptime monitoring systems
- Service discovery health verification

---

## Host Specific Configuration

In addition to global settings, each host has specific fields that control how it's monitored and woken.

### Static IP (static_ip)

Manually specify an IPv4 address for the host.

**Behavior:**

- When `use_as_fallback` is **false**: The system uses this IP directly for ARP ping, skipping the MAC-to-IP resolution phase. This is faster and more reliable if the device has a permanent, unchanging IP.
- When `use_as_fallback` is **true**: The system first tries to find the host's current IP by its MAC address. If not found, it falls back to this static IP.

**Use cases:**

- Devices with static IP assignments
- Complex networks where ARP scanning might be slow
- Fallback for devices that don't always respond to ARP scans

---

### Use as Fallback (use_as_fallback)

Toggle how the Static IP is used.

**Values:**

- `true` - Use Static IP only as a last resort if MAC resolution fails.
- `false` - Always use Static IP directly (default when IP is provided).

---

## Logging Configuration

Detailed logging settings (see [LOGGING.md](LOGGING.md) for full guide).

### log_level (string)

Verbosity of logs: `debug`, `info`, `warning`, `error`. (Default: `info`)

### log_output_mode (string)

Where logs are sent: `stdout`, `file`, `both`. (Default: `stdout`)

### log_dir (string)

Directory for log files when using `file` or `both` modes. (Default: `./logs`)

### log_rotation (boolean)

Enable automatic log rotation. (Default: `true`)

---

## Important Warnings

### Multiple Interfaces with Overlapping IP Ranges

**CRITICAL**: When using multiple network interfaces with `default_network_interface` or per-host interface selection, be aware of IP address conflicts:

**Problem Scenario:**

```
eth0: 192.168.1.0/24 - Device A at 192.168.1.100 (MAC: AA:BB:CC:DD:EE:FF)
eth1: 192.168.1.0/24 - Device B at 192.168.1.100 (MAC: 11:22:33:44:55:66)
```

**What Happens:**

- Ping operation tries eth0 first, finds IP 192.168.1.100, and returns success
- But this might be the WRONG device if you intended to ping Device B on eth1
- The system only verifies MAC address AFTER getting a response
- First successful ping wins, regardless of which network segment the target is on

**When This Happens:**

- Overlapping IP subnets across different network interfaces
- Complex network topologies (GRE tunnels, VLAN trunking, overlay networks)
- Virtual network interfaces with mirrored configurations
- Multi-datacenter setups with identical internal IP ranges

**Solutions:**

1. **Use non-overlapping IP ranges** (Recommended)
   - eth0: 192.168.1.0/24
   - eth1: 192.168.2.0/24

2. **Use per-host interface selection** (`enable_per_host_interfaces: true`)
   - Specify exact interface for each host to avoid ambiguity

3. **Use single interface per host** (Most reliable)
   - Assign one specific interface per host based on its network location

**Best Practices:**

- Design network topology with non-overlapping subnets when possible
- Use per-host interface configuration for complex multi-subnet environments
- Test ping functionality after configuration changes
- Monitor logs for MAC address mismatch warnings

---

## Network Interface Modes

The server supports interface selection through `enable_per_host_interfaces` setting:

### Per-Host Interface Disabled (default)

```json
{
  "enable_per_host_interfaces": false,
  "default_network_interface": "eth0"
}
```

- All hosts use `default_network_interface` setting
- If `default_network_interface` is empty, uses all available interfaces
- Simplest configuration
- Works in both auth and no-auth modes
- **Supports multiple interfaces** for automatic fallback

**Examples:**

- `default_network_interface: "eth0"` → All hosts use eth0
- `default_network_interface: "eth0,eth1"` → All hosts try eth0 first, then eth1
- `default_network_interface: "eth0,eth1,wlan0"` → All hosts try three interfaces in order
- `default_network_interface: ""` → All hosts use all available interfaces

**Multiple Interface Example:**

```json
{
  "enable_per_host_interfaces": false,
  "default_network_interface": "eth0,eth1,wlan0"
}
```

All hosts will automatically try eth0, then eth1, then wlan0 until one succeeds.

### Per-Host Interface Enabled

```json
{
  "enable_per_host_interfaces": true
}
```

- Each host can specify its own interface(s)
- If host has no interface specified, uses all available interfaces
- Does NOT fallback to `default_network_interface` setting
- Maximum flexibility for multi-subnet environments
- **Per-host interfaces support multiple interfaces** (comma-separated)
- Works in both auth and no-auth modes
- Disabled in `readonly_mode`

**Examples:**

**Allow per-host selection:**

```json
{
  "enable_per_host_interfaces": true
}
```

- Host with interface "eth0" → uses eth0
- Host with interface "eth0,eth1" → tries eth0 first, then eth1
- Host with interface "eth0,wlan0" → tries eth0 first, then wlan0 (useful for failover)
- Host with no interface → uses all available interfaces

**With readonly mode:**

```json
{
  "enable_per_host_interfaces": true,
  "readonly_mode": true,
  "default_network_interface": "eth0"
}
```

All hosts use eth0 (host-specific interfaces ignored in readonly mode)

---

## Environment Variables

All config options can be overridden via environment variables:

| Environment Variable         | Config Field               | Example     |
| ---------------------------- | -------------------------- | ----------- |
| `LISTEN_ADDRESS`             | listen_address             | `:8090`     |
| `URL_PREFIX`                 | url_prefix                 | `/wolweb`   |
| `DEFAULT_NETWORK_INTERFACE`  | default_network_interface  | `eth0,eth1` |
| `ENABLE_PER_HOST_INTERFACES` | enable_per_host_interfaces | `true`      |
| `PING_TIMEOUT_SECONDS`       | ping_timeout_seconds       | `10`        |
| `AUTH_EXPIRE_HOURS`          | auth_expire_hours          | `8`         |
| `USE_AUTH`                   | use_auth                   | `false`     |
| `READONLY_MODE`              | readonly_mode              | `true`      |
| `BEHIND_PROXY`               | behind_proxy               | `true`      |
| `DEBUG`                      | debug                      | `true`      |
| `HEALTH_CHECK_ENABLED`       | health_check_enabled       | `true`      |

**Example Docker usage:**

```bash
docker run -e LISTEN_ADDRESS=:8090 -e DEBUG=true -e BEHIND_PROXY=true ...
```

**Example systemd service:**

```ini
[Service]
Environment="LISTEN_ADDRESS=:8090"
Environment="DEBUG=true"
Environment="BEHIND_PROXY=true"
ExecStart=/usr/local/bin/wol-server -config /etc/wol/config.json
```

---

## Command-line Arguments

Override configuration for specific runtime scenarios:

```bash
# Show help
wol-server -h
wol-server --help

# Custom config and database paths
wol-server -config /etc/wol/config.json -db /var/lib/wol/data.db

# Enable debug mode (overrides config)
wol-server -debug

# Reset superuser password (interactive - select user and enter new password)
wol-server --reset-admin

# Combine options
wol-server -config custom.json -db data.db -debug
```

---

## Priority Order

Configuration values are loaded in this priority order (highest to lowest):

1. **Command-line flags** (`-debug`, `-config`, `-db`)
2. **Environment variables** (`LISTEN_ADDRESS`, `DEBUG`, etc.)
3. **Config file** (`config.json`)
4. **Default values**

Example:

```bash
# config.json has "debug": false
# But command-line flag overrides it:
wol-server -debug
# Result: Debug mode ENABLED
```

---

## Validation

The server validates configuration on startup:

- `listen_address` must be valid `address:port` format
- `ping_timeout_seconds` must be 1-60
- `auth_expire_hours` must be at least 1
- Network interfaces are checked on first use (not at startup)

**Example validation error:**

```
Invalid configuration: invalid listen_address format 'localhost': missing port in address
```

---

## Migration from Old Config

If you have an old config.json with `server_port` and `server_address`:

**Old format:**

```json
{
  "server_port": 8090,
  "server_address": "0.0.0.0"
}
```

**Automatic conversion:**
The server automatically converts this to:

```json
{
  "listen_address": "0.0.0.0:8090"
}
```

You'll see this log message:

```
INFO: Converted deprecated server_address:server_port to listen_address: 0.0.0.0:8090
```

**Action required:** Update your config.json to use `listen_address` directly.

---

## Security Best Practices

1. **Use HTTPS in production:**

   ```json
   {
     "behind_proxy": true,
     "use_auth": true
   }
   ```

2. **Bind to localhost if using reverse proxy:**

   ```json
   {
     "listen_address": "127.0.0.1:8090"
   }
   ```

3. **Enable auth for internet-facing deployments:**

   ```json
   {
     "use_auth": true
   }
   ```

4. **Use readonly mode for public kiosks:**

   ```json
   {
     "use_auth": false,
     "readonly_mode": true
   }
   ```

5. **Limit session duration:**
   ```json
   {
     "auth_expire_hours": 2
   }
   ```

---

## Troubleshooting

### "PERMISSION ERROR - Database file"

- Ensure read/write permissions on database file
- Check directory permissions
- Run as appropriate user (not root unless necessary)

### "invalid listen_address format"

- Use format `address:port` or `:port`
- Examples: `:8090`, `127.0.0.1:8090`
- Don't use `localhost:8090` (use `127.0.0.1:8090` instead)

### WoL packets not working

- Enable debug mode: `"debug": true`
- Check `default_network_interface` setting
- Verify broadcast address in host configuration
- Check logs for detailed error messages

### "operation not permitted" errors (ARP ping)

ARP ping and cache management require CAP_NET_RAW and CAP_NET_ADMIN capabilities:

**Docker:**

```bash
# Add capabilities in docker run
docker run --cap-add=NET_RAW --cap-add=NET_ADMIN ...

# Or in compose.yml
cap_add:
  - NET_RAW    # Required for ARP ping
  - NET_ADMIN  # Required for ARP cache flushing
```

**Bare-metal Linux:**

```bash
# Grant both capabilities to binary
sudo setcap cap_net_raw,cap_net_admin+ep /path/to/wol-server

# Verify capabilities
getcap /path/to/wol-server

# If needed, remove capabilities
sudo setcap -r /path/to/wol-server
```

**Without CAP_NET_RAW:**

- ARP ping and network scanning will be disabled
- Warning messages will appear: "operation not permitted"
- Basic WoL functionality still works
- ICMP ping still works (doesn't require capabilities)

### Debug mode not working

- Priority: CLI flag > env var > config file
- Try: `wol-server -debug` to force enable
- Check logs for `DEBUG MODE ENABLED` message

---

## Examples

### Minimal config (defaults)

```json
{
  "listen_address": ":8090"
}
```

### Development config

```json
{
  "listen_address": "127.0.0.1:3000",
  "use_auth": false,
  "debug": true
}
```

### Production config (behind Nginx)

```json
{
  "listen_address": "127.0.0.1:8090",
  "url_prefix": "/wolweb",
  "use_auth": true,
  "behind_proxy": true,
  "auth_expire_hours": 8,
  "debug": false
}
```

### Kiosk mode

```json
{
  "listen_address": ":8090",
  "use_auth": false,
  "readonly_mode": true
}
```

### Multi-subnet Linux server

```json
{
  "listen_address": ":8090",
  "use_auth": true,
  "enable_per_host_interfaces": true,
  "default_network_interface": "eth0",
  "behind_proxy": true
}
```

### Multi-interface with automatic fallback

```json
{
  "listen_address": ":8090",
  "use_auth": true,
  "enable_per_host_interfaces": false,
  "default_network_interface": "eth0,eth1,wlan0",
  "behind_proxy": true
}
```

All hosts automatically try eth0, then eth1, then wlan0 for redundancy.

---

## See Also

- [CLAUDE.md](CLAUDE.md) - Development guide
- [README.md](README.md) - General documentation
- [config.example.json](config.example.json) - Example configuration
