---
name: device-management
description: Query devices (including filtering dynamically by online/offline/alarm state, all devices or scoped to one group) and device groups managed by DeviceLink/AppHub, and control individual devices - reboot, shutdown, set volume/backlight, enable/disable USB storage, trigger a time sync, add a device to a group, or delete a device. Use this whenever the user asks to look up, list, group, or remotely control an AppHub-managed device via the eim-skill CLI.
---

# Device management

Wraps DeviceLink/AppHub's device management REST API as `eim-skill device
...` / `eim-skill group ...` commands. Requires `eim-skill` to be built
(`make` in the repo root) and `conf/config.yaml` (or `--base-url
--username --password` flags / `EIM_*` env vars) pointing at a reachable
AppHub server.

## When to use this skill

- "List all devices" / "which devices are in group X" → `device list`
- "Which devices are online/offline right now" (all, or in group X) → `device list --online` / `--offline` [`--group X`]
- "Which devices have an alarm/warning" (all, or in group X) → `device list --alarm` [`--group X`]
- "What's the status/detail of device X" → `device get`
- "What are device X's cpu/memory/storage/network specs" → `device details`
- "Reboot / shut down device X" → `device reboot` / `device shutdown`
- "Turn the volume/screen brightness up/down on device X" → `device set-volume` / `device set-backlight`
- "Disable the USB port on device X" (data-leak-prevention style request) → `device usb set-storage`
- "Sync the clock on device X" → `device sync-time`
- "Add device X to group Y" / "delete device X" → `device add-to-group` / `device delete`
- "What device groups exist" → `group list`
- "Create a new device group" → `group create`

## Commands

```
eim-skill device list                                  # all devices
eim-skill device list --group <group-name>             # devices in one group
eim-skill device list --online                          # only online devices
eim-skill device list --offline                          # only offline devices
eim-skill device list --alarm                            # only devices with a software/hardware alarm
  [--group <group-name>]                                 # any filter above can combine with --group
eim-skill device get <endpoint>                        # full device record
eim-skill device details <endpoint>                    # agent-reported system info (cpu/memory/storage/network)
eim-skill device delete <endpoint>                     # delete a device
eim-skill device add-to-group <endpoint> <group>        # add to a group (keeps existing memberships)
eim-skill group list                                    # all device groups
eim-skill group create <name> <type> [description] [--creator <username>]
  # type: typically "linux", "windows", "android", or "mix"

eim-skill device reboot <endpoint>
eim-skill device shutdown <endpoint>
eim-skill device set-volume <endpoint> <0-100>
eim-skill device set-backlight <endpoint> <0-100>
eim-skill device usb get-storage <endpoint>              # is USB mass storage enabled?
eim-skill device usb set-storage <endpoint> <on|off>
eim-skill device sync-time <endpoint>
```

`<endpoint>` is the device's endpoint id/MAC, as shown by `device list`.

## Notes and caveats

- `--online`, `--offline`, and `--alarm` are mutually exclusive on `device
  list` (each is a distinct server-side filter); `--group` can combine with
  any one of them. "Alarm" means the device's combined status
  (`deviceStatus`) is not 0/normal - it's 1 (error) or 2 (warning) on
  either the software or hardware side.
- `set-volume`, `set-backlight`, `usb set-storage`, and `sync-time` are not
  documented in AppHub-v2.3.0.md; they're inferred from DeviceLink's
  `PluginsServlet` Java source (the same devicecontrol resource used for
  the documented reboot/shutdown actions). Verify against real device
  firmware before relying on them in an unattended workflow.
- `device details` is also undocumented (found in `ClientServlet`'s
  `get_1_details`) - it's a lighter-weight call than `device get` when you
  only need the system-info blob, not the full device record (online
  status, group membership, etc).
- All commands print JSON to stdout and a non-zero exit code + stderr
  message on failure - safe to script against.
- See the top-level README for full configuration details (multiple
  backend servers, env var names, etc).
