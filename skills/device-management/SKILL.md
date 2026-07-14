---
name: device-management
description: Query devices (including filtering dynamically by online/offline/alarm state, all devices or scoped to one group) and device groups managed by DeviceLink/AppHub, and control individual devices - reboot, shutdown, set volume/backlight, enable/disable USB storage, trigger a time sync, add a device to a group, or delete a device. Use this whenever the user asks to look up, list, group, or remotely control an AppHub-managed device via the eim-skill CLI. 中文触发词：设备管理、设备列表、查询设备、在线设备、离线设备、告警设备、设备组、创建设备组、添加设备到组、删除设备、重启设备、关机、设置音量、设置背光/亮度、禁用U盘/USB存储、时间同步。
---

# Device management

Wraps DeviceLink/AppHub's device management REST API as `eim-skill device
...` / `eim-skill group ...` commands. Requires `eim-skill` to be built
(`make` in the repo root) and `conf/config.yaml` (or `--base-url
--username --password` flags / `EIM_*` env vars) pointing at a reachable
AppHub server.

## When to use this skill

- "List all devices" / "which devices are in group X" / "查询所有设备" / "查看X设备组下的设备" → `device list`
- "Which devices are online/offline right now" (all, or in group X) / "哪些设备在线/离线" → `device list --online` / `--offline` [`--group X`]
- "Which devices have an alarm/warning" (all, or in group X) / "哪些设备有告警" / "查询告警设备" → `device list --alarm` [`--group X`]
- "How many devices are online/offline" / "在线设备数和离线设备数" → `device counts`
- "What's the online devices' health breakdown (normal/warning/error)" / "在线设备的状态" → `device health`
- "What's the status/detail of device X" / "查看设备X的详情" → `device get`
- "What are device X's cpu/memory/storage/network specs" / "查看设备X的CPU/内存/存储/网络信息" → `device details`
- "Reboot / shut down device X" / "重启设备X" / "关闭设备X" / "设备关机" → `device reboot` / `device shutdown`
- "Turn the volume/screen brightness up/down on device X" / "设置设备X的音量" / "设置设备X的背光/亮度" → `device set-volume` / `device set-backlight`
- "Reboot / shut down every device in group Y" / "重启设备组Y下所有设备" / "给设备组Y批量关机" → `device reboot-group` / `device shutdown-group`
- "Set the volume/brightness for every device in group Y" / "给设备组Y批量设置音量/背光" → `device set-volume-group` / `device set-backlight-group`
- "Disable the USB port on device X" (data-leak-prevention style request) / "禁用设备X的U盘/USB存储" → `device usb set-storage`
- "Sync the clock on device X" / "同步设备X的时间" → `device sync-time`
- "Add device X to group Y" / "delete device X" / "把设备X添加到设备组Y" / "删除设备X" → `device add-to-group` / `device delete`
- "What device groups exist" / "查询所有设备组" → `group list`
- "How many online/warning/error devices does group X have" / "指定设备组下有几个在线设备，几个告警设备" → `group status X`
- "Create a new device group" / "创建设备组" / "添加设备组" → `group create`

## Commands

```
eim-skill device list                                  # all devices
eim-skill device list --group <group-name>             # devices in one group
eim-skill device list --online                          # only online devices
eim-skill device list --offline                          # only offline devices
eim-skill device list --alarm                            # only devices with a software/hardware alarm
  [--group <group-name>]                                 # any filter above can combine with --group
eim-skill device counts                                  # online/offline device counts
eim-skill device health                                  # online devices' status breakdown (normal/warning/error)
eim-skill device get <endpoint-or-name>                        # full device record
eim-skill device details <endpoint-or-name>                    # agent-reported system info (cpu/memory/storage/network)
eim-skill device delete <endpoint-or-name>                     # delete a device
eim-skill device add-to-group <endpoint-or-name> <group>        # add to a group (keeps existing memberships)
eim-skill group list                                    # all device groups
eim-skill group status [group-name]                     # groups' online/warning/error breakdown (all, or one)
eim-skill group create <name> <type> [description] [--creator <username>]
  # type: typically "linux", "windows", "android", or "mix"

eim-skill device reboot <endpoint-or-name>
eim-skill device shutdown <endpoint-or-name>
eim-skill device set-volume <endpoint-or-name> <0-100>
eim-skill device set-backlight <endpoint-or-name> <0-100>
eim-skill device reboot-group <group-name> [--label <name>]         # reboot every device in a group
eim-skill device shutdown-group <group-name> [--label <name>]       # shut down every device in a group
eim-skill device set-volume-group <group-name> <0-100> [--label <name>]
eim-skill device set-backlight-group <group-name> <0-100> [--label <name>]
eim-skill device usb get-storage <endpoint-or-name>              # is USB mass storage enabled?
eim-skill device usb set-storage <endpoint-or-name> <on|off>
eim-skill device sync-time <endpoint-or-name>
```

`<endpoint-or-name>` accepts either the device's endpoint id/MAC or its
human-readable device name (`devName`), both as shown by `device list` -
every device-targeting command resolves it against the live device list.

## Notes and caveats

- Every device object from `device list`/`device get` includes
  `deviceStatusText` (`"normal"`/`"warning"`/`"error"`) alongside the raw
  numeric `deviceStatus` (0/1/2, the same code `device list --alarm`
  filters on) - translated so you don't need to know the mapping.
- `device counts`/`device health`/`group status` are documented in
  AppHub-v2.3.0.md but the server returns a flat `[{"name":...,"value":...}]`
  array (`"gname"` instead of `"name"` for `group status`); this CLI
  normalizes them into one object with clear field names.
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
- `reboot-group`/`shutdown-group`/`set-volume-group`/`set-backlight-group`
  target every device in a group in one call (documented in
  AppHub-v2.3.0.md as "PUT send command-向批量设备发送命令",
  `/api/plugins/batch/{groupname}/40008/0/27601`) rather than looping over
  `device list --group` and calling the single-device command per device.
  `--label` optionally narrows the target to devices carrying that label.
- `device details` is also undocumented (found in `ClientServlet`'s
  `get_1_details`) - it's a lighter-weight call than `device get` when you
  only need the system-info blob, not the full device record (online
  status, group membership, etc).
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match - safe to script against.
- See the top-level README for full configuration details (env var names, etc).
