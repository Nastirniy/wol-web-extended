# Logging Guide

Configure how WoL-Web records events, errors, and access logs.

## Quick Start (3 Scenarios)

### 1. Development (Default)

See logs directly in your terminal.

```bash
# Run with debug flag
./wol-server -debug
```

### 2. Production (File System)

Save logs to files with automatic rotation (recommended for systemd/bare metal).

**Config (`config.json`):**

```json
{
  "log_level": "info",
  "log_output_mode": "file",
  "log_dir": "/var/log/wol-web",
  "log_rotation": true
}
```

### 3. Docker

Let Docker handle the logs via Standard Output.

**Environment Variables:**

```bash
LOG_LEVEL=info
LOG_OUTPUT_MODE=stdout
```

## Production Environment (Linux/Systemd)

```bash
export LOG_LEVEL=info
export LOG_OUTPUT_MODE=file
export LOG_DIR=/var/log/wol-web
./wol-server
```

## Docker Compose

```yaml
services:
  wol:
    environment:
      - LOG_LEVEL=info
      - LOG_OUTPUT_MODE=stdout
```

## Configuration Reference

Use `config.json` or Environment Variables (Env vars take precedence).

| Setting       | Env Variable      | Default  | Description                              |
| ------------- | ----------------- | -------- | ---------------------------------------- |
| **Level**     | `LOG_LEVEL`       | `info`   | `debug`, `info`, `warning`, `error`      |
| **Output**    | `LOG_OUTPUT_MODE` | `stdout` | `stdout`, `file`, `both`                 |
| **Directory** | `LOG_DIR`         | `./logs` | Path to store log files (if `file` mode) |
| **Rotation**  | `LOG_ROTATION`    | `true`   | Enable/disable automatic file rotation   |

### Advanced Rotation Settings

| Setting      | Env Variable       | Default | Description                        |
| ------------ | ------------------ | ------- | ---------------------------------- |
| **Max Size** | `LOG_MAX_SIZE_MB`  | `100`   | Rotate after this size (MB)        |
| **Max Age**  | `LOG_MAX_AGE_DAYS` | `30`    | Delete logs older than this (days) |

### Configuration Matrix

| Feature    | JSON Key           | Env Var            | Values                           |
| ---------- | ------------------ | ------------------ | -------------------------------- |
| **Level**  | `log_level`        | `LOG_LEVEL`        | `debug`, `info`, `warn`, `error` |
| **Mode**   | `log_output_mode`  | `LOG_OUTPUT_MODE`  | `stdout`, `file`, `both`         |
| **Path**   | `log_dir`          | `LOG_DIR`          | e.g., `/var/log/wol-web`         |
| **Rotate** | `log_rotation`     | `LOG_ROTATION`     | `true`, `false`                  |
| **Size**   | `log_max_size_mb`  | `LOG_MAX_SIZE_MB`  | e.g., `50`                       |
| **Age**    | `log_max_age_days` | `LOG_MAX_AGE_DAYS` | e.g., `30`                       |

## Log Levels

1. **DEBUG**: Verbose details (ARP scanning, network packets). Use for troubleshooting.
2. **INFO**: Standard events (startup, user login, wake requests). Use for normal operation.
3. **WARNING**: Non-critical issues (config warnings).
4. **ERROR**: Critical failures (database errors, port blocked).

## Need Help?

- Check `apps/server/config.go` for implementation details.
- **No logs?** Check `LOG_LEVEL` isn't set to `error`.
- **Permission denied?** Run `chown -R user:user /path/to/logs`.
- **Disk full?** Reduce `LOG_MAX_AGE_DAYS` or `LOG_MAX_SIZE_MB`.
