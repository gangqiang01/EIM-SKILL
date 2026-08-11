---
name: device-management
description: Query devices (including filtering dynamically by online/offline/alarm state, all devices or scoped to one group) and device groups managed by DeviceOn-EIM, control individual devices - reboot, shutdown, set volume/backlight, trigger a time sync, add a device to a group, or delete a device - and get the server's own client-onboarding config for pairing new device agents. Use this whenever the user asks to look up, list, group, or remotely control a DeviceOn-EIM-managed device, or wants to connect/pair a new device agent, via the eim-skill CLI. 中文触发词：设备管理、设备列表、查询设备、在线设备、离线设备、告警设备、设备组、创建设备组、添加设备到组、删除设备、重启设备、关机、设置音量、设置背光/亮度、时间同步、设备配对、二维码配置、客户端连接。
---

# Device management

Wraps DeviceOn-EIM's device management REST API as `eim-skill device
...` / `eim-skill group ...` commands. Requires `eim-skill` to be built
(`make` in the repo root) and `conf/config.yaml` (or `--base-url
--username --password` flags / `EIM_*` env vars) pointing at a reachable
DeviceOn-EIM server. If the server's base URL is `https://` with a
certificate from an internal/private CA, an `x509: certificate signed by
unknown authority` error means you need `ca_cert_file`/`--ca-cert-file`
or `insecure_skip_verify`/`--insecure` (see repo README's "TLS trust"
section).

## Running the eim-skill binary / 如何运行 eim-skill 二进制

Commands below are written as `../scripts/eim-skill ...`, assuming a
Linux x86_64 host and this file's own directory as the working
directory - the binary ships in a `scripts/` folder that sits alongside
every skill folder (i.e. one level up from this file, at
`../scripts/`), not on PATH. On a different host, substitute the
matching binary name everywhere `../scripts/eim-skill` appears below:

- Linux arm64/aarch64: `../scripts/eim-skill-arm64`
- Windows: `../scripts/eim-skill.exe`

下面的命令统一写成 `../scripts/eim-skill ...`，默认假设运行在 Linux
x86_64 主机上，且执行时的工作目录就是这个文件所在的目录——二进制文件
放在和每个技能目录同级的 `scripts/` 目录下（即这个文件所在目录的
上一级，路径为 `../scripts/`），而不在 PATH 里。如果运行在其他平台，
请把下面所有出现的 `../scripts/eim-skill` 替换成对应的二进制名：

- Linux arm64/aarch64：`../scripts/eim-skill-arm64`
- Windows：`../scripts/eim-skill.exe`

## When to use this skill

- "List all devices" / "which devices are in group X" / "查询所有设备" / "查看X设备组下的设备" → `device list`
- "Which devices are online/offline right now" (all, or in group X) / "哪些设备在线/离线" → `device list --online` / `--offline` [`--group X`]
- "Which devices have an alarm/warning" (all, or in group X) / "哪些设备有告警" / "查询告警设备" → `device list --alarm` [`--group X`]
- "How many devices are online/offline" / "在线设备数和离线设备数" → `device counts`
- "What's the online devices' health breakdown (normal/warning/error)" / "在线设备的状态" → `device health`
- "What's the status/detail of device X" / "查看设备X的详情" → `device get`
- "What are device X's cpu/memory/storage/network specs" / "查看设备X的CPU/内存/存储/网络信息" → `device details`
- "Reboot / shut down device X" / "重启设备X" / "关闭设备X" / "设备关机" → confirm with the user first, then `device reboot` / `device shutdown`
- "Schedule device X to reboot/shut down at a future time, weekly, or monthly" / "设置设备X定时/每周/每月重启或关机" → confirm with the user first, then `device schedule reboot` / `device schedule shutdown` (`--at`/`--weekly`/`--monthly`)
- "Cancel the scheduled reboot/shutdown on device X" / "取消设备X的定时重启/关机" → `device schedule reboot --cancel` / `device schedule shutdown --cancel`
- "What's the current scheduled reboot/shutdown for device X?" / "查看设备X当前的定时重启/关机任务" → `device schedule get-reboot` / `device schedule get-shutdown`
- "Take a screenshot of device X (or devices X, Y, Z)" / "给设备X截屏" / "给X、Y、Z这几台设备截屏" → `device screenshot` (pass multiple devices in one call rather than looping)
- "Turn the volume/screen brightness up/down on device X" / "设置设备X的音量" / "设置设备X的背光/亮度" → `device set-volume` / `device set-backlight`
- "What's device X's current volume/backlight set to?" / "查看设备X当前的音量/背光" → `device get-volume` / `device get-backlight`
- "Reboot / shut down every device in group Y" / "重启设备组Y下所有设备" / "给设备组Y批量关机" → confirm with the user first (state how many/which devices this affects), then `device reboot-group` / `device shutdown-group`
- "Set the volume/brightness for every device in group Y" / "给设备组Y批量设置音量/背光" → `device set-volume-group` / `device set-backlight-group`
- "Sync the clock on device X" / "同步设备X的时间" → `device sync-time`
- "Add device X to group Y" / "delete device X" / "把设备X添加到设备组Y" / "删除设备X" → `device add-to-group` / `device delete`
- "What device groups exist" / "查询所有设备组" → `group list`
- "How many online/warning/error devices does group X have" / "指定设备组下有几个在线设备，几个告警设备" → `group status X`
- "Create a new device group" / "创建设备组" / "添加设备组" → `group create`
- "How do I connect/pair a new device to the server" / "怎么把新设备连接/配对到服务器" → `server get-config`

## Commands

```
../scripts/eim-skill device list                                  # all devices
../scripts/eim-skill device list --group <group-name>             # devices in one group
../scripts/eim-skill device list --online                          # only online devices
../scripts/eim-skill device list --offline                          # only offline devices
../scripts/eim-skill device list --alarm                            # only devices with a software/hardware alarm
  [--group <group-name>]                                 # any filter above can combine with --group
../scripts/eim-skill device counts                                  # online/offline device counts
../scripts/eim-skill device health                                  # online devices' status breakdown (normal/warning/error)
../scripts/eim-skill device get <endpoint-or-name>                        # full device record
../scripts/eim-skill device details <endpoint-or-name>                    # agent-reported system info (cpu/memory/storage/network)
../scripts/eim-skill device delete <endpoint-or-name>                     # delete a device
../scripts/eim-skill device add-to-group <endpoint-or-name> <group>        # add to a group (keeps existing memberships)
../scripts/eim-skill group list                                    # all device groups
../scripts/eim-skill group status [group-name]                     # groups' online/warning/error breakdown (all, or one)
../scripts/eim-skill group create <name> <type> [description] [--creator <username>]
  # type: typically "linux", "windows", "android", or "mix"

../scripts/eim-skill device reboot <endpoint-or-name>
../scripts/eim-skill device shutdown <endpoint-or-name>
../scripts/eim-skill device schedule reboot <endpoint-or-name> --at <datetime>          # schedule a future reboot
  | --weekly <mon..sun> --time <HH:MM>                              # ...or repeating weekly
  | --monthly <1-31> --time <HH:MM>                                 # ...or repeating ~monthly
  | --cancel                                                        # cancel a pending schedule
../scripts/eim-skill device schedule shutdown <endpoint-or-name> ...           # same flags, for shutdown
../scripts/eim-skill device schedule get-reboot <endpoint-or-name>            # what's currently scheduled (reboot)
../scripts/eim-skill device schedule get-shutdown <endpoint-or-name>          # what's currently scheduled (shutdown)
../scripts/eim-skill device screenshot <endpoint-or-name> [<endpoint-or-name>...]      # capture and save a screenshot
  [--interval <ms>] [--out-dir <dir>] [--host <url>]               # (batch-capable - pass multiple devices at once)
../scripts/eim-skill device get-volume <endpoint-or-name>
../scripts/eim-skill device set-volume <endpoint-or-name> <0-100>
../scripts/eim-skill device get-backlight <endpoint-or-name>
../scripts/eim-skill device set-backlight <endpoint-or-name> <0-100>
../scripts/eim-skill device reboot-group <group-name> [--label <name>]         # reboot every device in a group
../scripts/eim-skill device shutdown-group <group-name> [--label <name>]       # shut down every device in a group
../scripts/eim-skill device set-volume-group <group-name> <0-100> [--label <name>]
../scripts/eim-skill device set-backlight-group <group-name> <0-100> [--label <name>]
../scripts/eim-skill device sync-time <endpoint-or-name>

../scripts/eim-skill server get-config                                     # client-onboarding config blob (not device-scoped)
```

`<endpoint-or-name>` accepts either the device's endpoint id/MAC or its
human-readable device name (`devName`), both as shown by `device list` -
every device-targeting command resolves it against the live device list.

## Notes and caveats

- **Always confirm with the user before running `device reboot`,
  `device shutdown`, `device reboot-group`, `device shutdown-group`,
  `device schedule reboot`, or `device schedule shutdown`.** These
  interrupt a running device (or every device in a group) immediately or
  at a future time you've now armed, take effect whether or not anything
  important is running on it, and can't be undone once issued (or, for
  `schedule`, once the scheduled time arrives). Before calling any of
  them, state plainly what you're about to do and to which device(s) -
  for the `-group` variants, resolve and state how many/which devices
  are affected (e.g. via `device list --group <name>`); for `schedule`,
  state the resolved date/time (and repeat pattern, if any) - and wait
  for explicit confirmation. Do not reboot/shut down as a side effect of
  a broader request unless the user asked for exactly that.
- `device schedule get-reboot`/`get-shutdown` (via the
  `get_reboot_plan`/`get_shutdown_plan` devicecontrol actions) confirm
  that `device schedule reboot`/`shutdown` genuinely stores the
  schedule server-side: populated right after the write, and back to
  empty after `--cancel` - including a `--cancel` that used different
  `--at`/`--weekly`/`--monthly` values than the schedule it cleared, so
  `--cancel` doesn't need to match the original values exactly. The one
  thing not verified end-to-end is whether the device actually fires the
  reboot/shutdown at the scheduled time. `--monthly` is only an
  approximation (fires every ~30 days from the first run, not on a true
  calendar day-of-month) and will drift off the chosen day over time -
  `--weekly` doesn't have this problem (always exactly 7 days). Prefer
  `--weekly` over `--monthly` when either would do, and recommend the
  user test on a non-critical device before relying on this unattended.
- `device screenshot` accepts multiple devices in one call and captures
  them all together - prefer that over looping the command once per
  device. It's a two-step process (batch "start pushing screenshots"
  call, wait `--interval` ms, then fetch+save per device) confirmed
  end-to-end on a live Windows device. A device without an active
  display session (confirmed on a live headless Linux device) reports an
  error for just that device rather than failing the whole batch - check
  each result's `status` field, don't assume all devices succeeded just
  because the command exited 0.
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
- `get-volume`/`set-volume` and `get-backlight`/`set-backlight` use the
  `get_audio`/`set_audio` and `get_brightness`/`set_brightness`
  devicecontrol actions (same resource as the documented reboot/shutdown
  actions), all confirmed against a live server - an earlier version of
  `set-volume`/`set-backlight` used the wrong keys `audio`/`brightness`,
  which returned success without actually changing anything, fixed via a
  real bug report. `get-volume`/`get-backlight` reflect whatever the
  matching `set-*` command last recorded server-side - not confirmed
  whether that always matches the actual physical hardware value on every
  device model. `sync-time` is not documented in AppHub-v2.3.0.md and
  hasn't been verified against real
  firmware - verify before relying on it in an unattended workflow.
- `set-volume`/`set-backlight`/`set-volume-group`/`set-backlight-group`
  reject values outside 0-100 client-side - the server accepts and
  returns success for out-of-range values without erroring (confirmed via
  a real bug report), so this CLI validates before sending the request.
- `add-to-group` checks that group-name exists (via `group list`) before
  adding the device - the underlying server endpoint otherwise silently
  accepts and records a nonexistent group name (confirmed via a real bug
  report). It also checks that the device's OS matches the group's
  `type` (`linux`/`windows`/`android`/`openharmony` are the server's
  built-in defaults; a `mix` group accepts any OS) - the underlying
  endpoint enforces none of this either, confirmed via a real bug report
  that it silently succeeds adding a Windows device to a Linux-typed
  group.
- `reboot-group`/`shutdown-group`/`set-volume-group`/`set-backlight-group`
  target every device in a group in one call, rather than looping over
  `device list --group` and calling the single-device command per device.
  They resolve the group (and `--label`, if given) to device endpoints
  client-side, then execute against all of them via AppHub-v2.3.0.md's
  documented batch-execute resource (`/api/plugins/batchExec/40008/0/27601`,
  body `{"clientIds":[...],"value":...}`). `--label` optionally narrows
  the target to devices carrying that label. (A simpler-looking documented
  endpoint that takes the group name directly in the URL,
  `/api/plugins/batch/{groupname}/40008/0/27601`, was tried first but
  reliably returns an empty-bodied HTTP 500 on real servers - confirmed via
  a real bug report and reproduced independently - so it's not used.)
- `device details` is also undocumented (found in `ClientServlet`'s
  `get_1_details`) - it's a lighter-weight call than `device get` when you
  only need the system-info blob, not the full device record (online
  status, group membership, etc). Its shape varies by device OS and is
  mostly passed through as-is, except `memorysize` and each storage
  entry's `total` - the agent reports these as numeric strings (e.g.
  `"memorysize":"7.64"`) on both Linux and Windows devices, confirmed
  against a live server, so this CLI converts them to real JSON numbers.
- `server get-config` isn't scoped to a device - it returns the server's
  own client-onboarding config blob (`{"id":"WP.SRP","data":"<encrypted
  blob>"}`). Paste `data` directly into a device agent's own
  configuration UI to connect it to this server - this mirrors the web
  console's QR-code device-pairing feature (the same blob is what gets
  encoded into the QR code shown there). Not documented in
  AppHub-v2.3.0.md; confirmed against a live server.
- Every command prints one JSON object to stdout:
  `{"status":200,"msg":"","data":...}` or `{"status":500,"msg":"..."}`, with
  exit code 0/1 to match - safe to script against.
- See the top-level README for full configuration details (env var names, etc).
