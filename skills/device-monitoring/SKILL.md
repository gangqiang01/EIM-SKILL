---
name: device-monitoring
description: Read and configure a DeviceLink/AppHub device's monitoring and alerting - CPU/memory/storage/network/battery thresholds, application/process monitoring, Docker container monitoring, USB peripheral change monitoring, and historical CPU/memory samples. Use this whenever the user asks about a device's resource usage, alert thresholds, or wants to turn monitoring of some subsystem on/off via the eim-skill CLI.
---

# Device monitoring and alerting

Wraps DeviceLink/AppHub's per-device monitoring config, which is
threshold/alert based (not a live streaming dashboard) - this mirrors the
web console's "monitorAlert" feature. Requires `eim-skill` built and
configured (see repo README).

## When to use this skill

- "Is CPU/memory/storage/network monitoring on for device X, and what are
  the alert thresholds?" → `device threshold get`
- "Alert me if CPU usage on device X goes above 90%" → `device threshold set`
- "Turn hardware monitoring on/off for device X" → `device set-monitor`
- "What processes are running on device X, and is process monitoring on?" → `device process list` / `device process set-monitor`
- "What Docker containers are on device X?" → `device docker info`
- "Turn on USB-change alerts for device X" → `device usb set-monitor`
- "Show me CPU/memory history for device X" → `device monitor --type cpu|memory`

## Commands

```
eim-skill device set-monitor <endpoint> <on|off>                # master hw-monitoring toggle

eim-skill device threshold get <endpoint>
eim-skill device threshold set <endpoint> <metric> <on|off> [--warn N] [--error N]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process list <endpoint>
eim-skill device process set-monitor <endpoint> <on|off>

eim-skill device docker info <endpoint>
eim-skill device docker set-monitor <endpoint> <on|off>

eim-skill device usb get-monitor <endpoint>                      # USB change-alerting, distinct from
eim-skill device usb set-monitor <endpoint> <on|off>             # usb set-storage (mass-storage enable/disable)

eim-skill device monitor <endpoint> --type cpu|memory [--begin <unix-ms>] [--end <unix-ms>]
```

## Notes and caveats

- CPU/memory/storage/network/battery are monitored via configurable
  warn/error *thresholds*, not a live value feed. Pair `device threshold
  get` (current config) with `device monitor --type cpu|memory`
  (historical samples) - storage/network have no persisted history
  server-side.
- Process and Docker monitoring here are on/off toggles only; the web
  console's per-process/per-container selection (pick exactly which
  processes/containers to watch) isn't wired up in this CLI.
- `device usb get-monitor`/`set-monitor` (USB *change* alerting) is a
  different feature from `device usb get-storage`/`set-storage` (whether
  USB mass storage itself is enabled) - see the device-management skill
  for the latter.
