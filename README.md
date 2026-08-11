# EIM-SKILL

[中文文档](README-zh.md)

`eim-skill` is a command-line skill tool that wraps
[DeviceOn-EIM](../../../MYPROJECT/BACKEND/DeviceOn-EIM)'s device management
REST API (documented in `/opt/DeviceOn-EIM-v2.3.0.md`) so device operations can be
driven from the command line or invoked as a skill by an agent.

## Output format

Every command prints exactly one JSON object to stdout, regardless of
which command or whether it succeeded:

```json
{"status": 200, "msg": "", "data": ...}
{"status": 500, "msg": "..."}
```

The process exit code is `0` on success and `1` on failure - scripts/agents
can check either the exit code or the `status` field (`200`=success,
`500`=failure). `msg` is empty on success and holds the error text on
failure. `data`'s shape depends on the command (a device list, a status
object, `null`/omitted for actions with nothing to report, etc) and is
omitted entirely on failure.

Every command's `--help` (and its subcommands' `--help`) includes a
detailed description, parameter meanings, an example invocation, and
example output - in both English and Chinese (中文). Run
`eim-skill <command> --help` (or drill into a subcommand, e.g.
`eim-skill device set-volume --help`) for specifics beyond this README's
summary tables.

## Supported commands

```
eim-skill device list                                             # list all devices
eim-skill device list --group <group-name>                        # list devices in a group
eim-skill device list --online|--offline                          # filter by online status
eim-skill device list --alarm                                      # devices with a software/hardware alarm
  [--group <group-name>]                                           # any of the above can combine with --group
eim-skill device counts                                             # online/offline device counts
eim-skill device health                                             # online devices' status breakdown (normal/warning/error)
eim-skill device get <endpoint-or-name>                            # get the full device record
eim-skill device details <endpoint-or-name>                        # get device system info (cpu/memory/storage/network)
eim-skill device delete <endpoint-or-name>                         # delete a device
eim-skill device add-to-group <endpoint-or-name> <group>           # add a device to a group
eim-skill device monitor <endpoint-or-name> --type cpu|memory      # get cpu/memory monitoring data
  [--last <30m|12h|3d>]                                            # optionally restrict to a recent time window
  [--begin <time>] [--end <time>]                                  # or an absolute range instead of --last
eim-skill device get-monitor-report <endpoint-or-name>             # is cpu/memory historical-data collection on?
eim-skill device set-monitor-report <endpoint-or-name> <cpu|memory> <on|off>  # turn cpu/memory data collection on/off
eim-skill device reboot <endpoint-or-name>                         # reboot a device now
eim-skill device shutdown <endpoint-or-name>                       # shut down a device now
eim-skill device schedule reboot <endpoint-or-name> --at <datetime>          # schedule a future reboot
  | --weekly <mon..sun> --time <HH:MM>                              # ...or repeating weekly
  | --monthly <1-31> --time <HH:MM>                                 # ...or repeating ~monthly
  | --cancel                                                        # cancel a pending schedule
eim-skill device schedule shutdown <endpoint-or-name> ...           # same flags, for shutdown
eim-skill device schedule get-reboot <endpoint-or-name>            # what's currently scheduled (reboot)
eim-skill device schedule get-shutdown <endpoint-or-name>          # what's currently scheduled (shutdown)
eim-skill device screenshot <endpoint-or-name> [<endpoint-or-name>...]      # capture and save a screenshot
  [--interval <ms>] [--out-dir <dir>] [--host <url>]               # (batch-capable - pass multiple devices at once)
eim-skill device get-volume <endpoint-or-name>                     # get last-recorded audio volume
eim-skill device set-volume <endpoint-or-name> <0-100>             # set audio volume
eim-skill device get-backlight <endpoint-or-name>                  # get last-recorded screen backlight
eim-skill device set-backlight <endpoint-or-name> <0-100>          # set screen backlight
eim-skill device reboot-group <group-name> [--label <name>]        # reboot every device in a group
eim-skill device shutdown-group <group-name> [--label <name>]      # shut down every device in a group
eim-skill device set-volume-group <group-name> <0-100> [--label <name>]     # set audio volume on every device in a group
eim-skill device set-backlight-group <group-name> <0-100> [--label <name>] # set screen backlight on every device in a group
eim-skill group list                                               # list all device groups
eim-skill group status [group-name]                                # groups' online/warning/error breakdown (all, or one)
eim-skill group create <name> <type> [description] [--creator <username>]  # add a device group
```

`<endpoint-or-name>` accepts either the device's endpoint id (its MAC-like
id, as shown by `device list`) or its human-readable device name
(`devName`) - every device-targeting command resolves it against the live
device list, so device name always works as an alternative to the endpoint.

Every device object returned by `device list`/`device get` includes
`deviceStatusText` (`"normal"`, `"warning"`, or `"error"`) alongside the
raw numeric `deviceStatus` (0/1/2) - the same code `device list --alarm`
filters on, translated so you don't have to know the mapping.

`get-volume`/`set-volume` and `get-backlight`/`set-backlight` are not
documented in DeviceOn-EIM-v2.3.0.md; they use the same devicecontrol
resource as reboot/shutdown, with action keys `get_audio`/`set_audio` and
`get_brightness`/`set_brightness`, all confirmed against a live server (an
earlier version of the `set-*` commands used the wrong keys
`audio`/`brightness`, which returned success without actually changing
anything - fixed via a real bug report). `get-volume`/`get-backlight`
reflect whatever the matching `set-*` command last recorded server-side -
not confirmed whether that always matches the actual physical hardware
value on every device model. `set-volume`/`set-backlight`/
`set-volume-group`/`set-backlight-group` reject values outside 0-100
client-side - the server accepts and returns success for out-of-range
values without erroring, confirmed via a real bug report.

`add-to-group` checks that `<group>` exists (via `group list`) before
adding the device - the underlying server endpoint otherwise silently
accepts and records a nonexistent group name, confirmed via a real bug
report. It also checks that the device's OS matches the group's type
(`linux`/`windows`/`android`/`openharmony` are the server's built-in
defaults; a `mix` group accepts any OS) - the underlying endpoint
enforces none of this either, confirmed via a real bug report that it
silently succeeds adding a Windows device to a Linux-typed group.

`reboot-group`/`shutdown-group`/`set-volume-group`/`set-backlight-group`
target every device in a group with a single call: they resolve the group
(and `--label`, if given) to device endpoints client-side, then execute
the action against all of them in one request to DeviceOn-EIM's
documented batch-execute resource (`PUT
/api/plugins/batchExec/40008/0/27601`, body `{"clientIds":[...],
"value":...}`). `--label` optionally narrows the target to devices
carrying that label.

An earlier implementation instead called a simpler-looking documented
endpoint that takes the group name directly in the URL (`PUT
/api/plugins/batch/{groupname}/40008/0/27601`), but that was found - via a
real bug report, reproduced independently against a second server - to
reliably fail with an empty-bodied HTTP 500 for any payload, so these
commands now resolve the group to specific devices first instead.

`device details` is also undocumented (found in `ClientServlet`'s
`get_1_details`); it's a lighter-weight call than `device get` when you
only need the agent-reported system info blob (cpu, memory, storage,
network interfaces) rather than the full device record (online status,
group membership, etc). Its shape varies by device OS and is mostly
passed through as-is, except `memorysize` and each storage entry's
`total` - the agent reports these as numeric strings (e.g.
`"memorysize":"7.64"`) on both Linux and Windows devices, confirmed
against a live server, so this CLI converts them to real JSON numbers.

`device counts`/`device health`/`group status` are documented in
DeviceOn-EIM-v2.3.0.md but return a flat `[{"name":...,"value":...}]` array
(or, for `group status`, `"gname"` instead of `"name"`); this CLI
normalizes them into a single object with clear field names.

`device schedule reboot`/`shutdown` arm a future or recurring
reboot/shutdown (`--at` for once, `--weekly`/`--monthly` to repeat,
`--cancel` to disarm), and `device schedule get-reboot`/`get-shutdown`
report what's currently armed. Confirmed against a live server: the
write returns `{"status":"CHANGED"}` on the same single-device write
path as `device reboot`/`shutdown`, with `time` (the anchor Unix-ms
timestamp) and an optional `cycle` (repeat interval in days) added, and
`get-reboot`/`get-shutdown` (via the `get_reboot_plan`/`get_shutdown_plan`
devicecontrol actions) confirm the schedule is genuinely stored -
populated right after the write, and back to empty after `--cancel`,
including a `--cancel` that used different `--at`/`--weekly`/`--monthly`
values than the schedule it cleared. None of this is documented for a
single device in DeviceOn-EIM-v2.3.0.md - only the batch/group write
variant is (`PUT /api/plugins/batchschedule/{groupname}/40008/0/27601`),
whose own example uses `cycle:7` for a weekly repeat, confirming `cycle`
is a day-count. `--weekly` maps to this exactly (`cycle=7`, anchored to
the chosen weekday); `--monthly` is only an approximation (`cycle=30`) -
since calendar months aren't a fixed number of days, the day it actually
fires on will drift away from the day you picked the longer the schedule
runs. The one thing not verified end-to-end is whether the device
actually fires the reboot/shutdown at the scheduled time - confirming
that would require letting an unattended reboot/shutdown actually occur
on a real device, which wasn't done here.

`device screenshot` accepts one or more devices in a single call and
captures all of them together, rather than one at a time - it's a
two-step, undocumented process (confirmed against a live server via a
real packet capture): a batch call (`PUT /api/devicemgr/batchExec`, a
different endpoint/body shape than the `batchExec` used by
`reboot-group`/etc) tells every listed device to start pushing periodic
screenshots to this server every `--interval` milliseconds (default
5000); after waiting that same interval, each device's most recent
screenshot is fetched (`GET /api/devicemgr/screenshotimg/{agentId}`) and
saved to `--out-dir` (default: current directory) as a `.jpg`/`.png`
based on the response's Content-Type. Verified end-to-end on a live
Windows device (a real 1920x1080 JPEG screenshot). A device without an
active display session - confirmed on a live headless Linux device -
reports an error (`"Get screen image error"`) for that device only,
without failing devices captured in the same batch call.

### Monitoring and alerting

```
eim-skill device get-monitor <endpoint-or-name>                      # cpu/memory/storage/battery/network snapshot

eim-skill device threshold get <endpoint-or-name>                   # current threshold config
eim-skill device threshold set <endpoint-or-name> <metric> <on|off> # configure one metric
  [--warn <n>] [--error <n>]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process list <endpoint-or-name>                    # every process on the device
eim-skill device process monitored <endpoint-or-name>               # processes currently selected for monitoring
eim-skill device process set-monitor <endpoint-or-name> <pid> <on|off>       # add/remove one process from monitoring

eim-skill device docker list <endpoint-or-name>                     # every Docker container on the device
eim-skill device docker monitored <endpoint-or-name>                 # docker containers currently selected for monitoring
eim-skill device docker set-monitor <endpoint-or-name> <container-name> <on|off>  # add/remove one container from monitoring

eim-skill device usb get-monitor <endpoint-or-name>                 # is USB change-alerting on?
eim-skill device usb set-monitor <endpoint-or-name> <on|off>        # USB change-alerting on/off
eim-skill device usb get-storage <endpoint-or-name>                 # is USB mass-storage device access enabled?
eim-skill device usb set-storage <endpoint-or-name> <on|off>        # USB mass-storage device access on/off
```

This mirrors the DeviceOn-EIM web console's "monitorAlert" feature: CPU,
memory, storage and network are monitored via configurable warn/error
thresholds (`device threshold`) rather than a live dashboard feed - pair
this with `device monitor --type cpu|memory` for historical CPU/memory
samples, or `device get-monitor` for a live snapshot (see below).

`device threshold set` only overwrites `--warn`/`--error` when you pass
them. If either is omitted (e.g. just toggling a metric on/off), it first
fetches that metric's current warn/error levels via `device threshold
get` and resends those unchanged - the underlying `set_threshold`
devicecontrol action requires both warn and error on every request, and
an earlier version sent `0` for whichever flag was omitted, which
silently zeroed out any existing threshold on every on/off toggle
(confirmed via a real bug report and packet capture).

`--warn`/`--error` are validated client-side, since the server accepts
anything - confirmed live that it silently stores `warn=150`/`error=200`,
or even negative values, verbatim for `cpu_usage`. Percentage metrics
(`cpu_usage`, `memory_usage`, `storage_usage`, `battery_available`) must
be within 0-100, neither may be negative, and the pair must be ordered
correctly for the metric's direction: `warn < error` everywhere except
`battery_available`, where lower is worse, so it's reversed (`warn >
error`, e.g. `warn=20, error=10`). A pure on/off toggle with no
`--warn`/`--error` skips these checks entirely.

There is no `device set-monitor` (overall hardware monitoring on/off) -
it used the `set_isMonitor` devicecontrol action and has been removed;
`device threshold set` already lets you disable alerting per-metric.
There is also no `device process set-monitor`/`device docker
set-monitor`: they used the `set_isProcessMonitor`/
`set_isDockerMonitor` devicecontrol actions and were confirmed against a
live server to return success without actually changing anything, so
they were removed rather than shipped non-functional.

`device process list`/`device docker list` return every process/Docker
container on a device regardless of monitor status; `device process
monitored`/`device docker monitored` filter to just the subset currently
selected for monitoring. Both pairs share the same underlying
devicecontrol actions (`get_software_info`, `get_docker_info`) -
`monitored` just filters `list`'s result client-side
(`is_monitored`/`ismonitor == true`) rather than making a separate
request. `device docker list`'s endpoint was confirmed via a real packet
capture; an earlier version instead read from an undocumented LwM2M
resource (`.../60001/0/27300`) assuming a response grouped by compose
project, which returned an empty result on a live server with real
containers deployed, so it was switched to the working `get_docker_info`
action.

`device process set-monitor`/`device docker set-monitor` add or remove a
single process (by pid) or container (by name) to/from that monitored
list - you only give the identifier; the underlying API actually
requires resending the *entire* process/container object (from `list`)
with an add/remove flag, so this CLI fetches the current list itself,
finds the matching item, and resends it verbatim. This is confirmed
against a live server via a real packet capture and cross-verified
end-to-end (adding a pid/container makes it appear in
`monitored`, removing it makes it disappear, without disturbing anything
else already monitored). There's no single-device version of this
documented in DeviceOn-EIM-v2.3.0.md for processes - only a batch/group
endpoint is (`PUT /api/plugins/batchsetsoftwaremonitor/{groupname}/
40008/0/27601`) - and that one reliably fails with an empty-bodied HTTP
500 on a live server (the same failure mode as another
documented-but-broken batch endpoint elsewhere in this CLI), so the
single-device path (undocumented, but shares the same request shape) is
used instead. Along the way, this also uncovered and fixed a decode bug:
`device process monitored`'s `is_monitored` had always reported `false`
even for genuinely monitored processes, because `get_monitor_app` never
actually populates the per-item `threadismonitor` field - being present
in that list at all is what "monitored" means for this action, so
`is_monitored` is now always `true` for every item it returns.

`device usb get-monitor`/`set-monitor` control whether USB peripheral
change-alerting is on (alert when a USB device is connected/
disconnected). They use the `get_usb_change`/`set_peripheral_monitor`
devicecontrol actions, confirmed against a live server via a real packet
capture and cross-verified by toggling `set-monitor` true/false and
observing `get-monitor` correctly reflects each change. (A brief earlier
version of this client swapped in `get_usb_storage_status`/
`set_usb_storage_status` instead, based on a packet capture that turned
out to be for a different, separate feature - USB mass-storage access,
not this one; that action's SET call succeeded and was logged under a
friendly audit-log event name, which looked like good evidence at the
time, but its GET side never reflected the change on a live device
tested, so it was reverted. A GET that actually reflects the SET is
stronger evidence of a real, working toggle than an audit-log label.)

`device usb get-storage`/`set-storage` are a separate, independent
switch from `get-monitor`/`set-monitor` above - whether the device
allows USB mass-storage devices (e.g. a USB flash drive) to be used, not
connect/disconnect alerting. `set-storage` is confirmed via the
`set_usb_storage_status` devicecontrol action, cross-checked in `log
operations list`'s audit trail (`on` logged as `enableUsb`, `off` as
`disableUsb`); `get-storage` uses `get_usb_storage_status` and was
cross-verified on a live Windows device by toggling `set-storage`
true/false and observing it correctly reflects each change in real
time. Not every device supports this action - a Linux industrial PC
tested returned an empty result regardless of the last value set, which
is normalized to `enabled: false` here (there's no way to distinguish
"genuinely off" from "not supported" via this action alone).

`device monitor`'s `value` is the usage percentage (0-100) for whichever
`--type` you asked for - CPU usage % for `--type cpu`, memory usage % for
`--type memory` (inferred from observed values matching
`device get-monitor`'s `cpu.usage_percent`/`memory.usage_percent`, not a
spec). It's parsed leniently (JSON number or numeric string, both
normalized) since some servers report it as a string.

`device get-monitor-report`/`set-monitor-report` control whether that
historical data is being collected at all - `device monitor --type
cpu|memory` has nothing new to show for a type that's off. Not
documented in AppHub-v2.3.0.md; `set-monitor-report` uses the
`set_monitor_data_report` devicecontrol action (confirmed via a real
packet capture) - the same action `log system set-monitor` uses for
system logs, just with a `cpu_monitor`/`memory_monitor` key instead of
`systemd_log_monitor`. `get-monitor-report` (`get_monitor_data_report`)
was found while investigating this and confirmed live (write, then
read back both before and after, for both cpu and memory
independently) - it also happens to report `systemd_log_monitor` in the
same response, which `log system get-monitor` now exposes separately.

`device get-monitor` is not documented in DeviceOn-EIM-v2.3.0.md (found by
inspecting real traffic to the `get_board_info` devicecontrol action). Its
response is normalized: the raw agent payload mixes numbers, numeric
strings, and `"N/A"` placeholders inconsistently, so this CLI parses it
into consistent `float64` fields (`null` when unavailable) with
self-documenting names (`usage_percent`, `total_gb`, etc). Units/semantics
are inferred from observed values, not a spec - see the `BoardInfo`
doc comment in `internal/DeviceOn-EIM/types.go`.

`device process monitored` is also undocumented (the `get_monitor_app`
devicecontrol action, found the same way) and normalized similarly -
`threadcpuloading`/`threadmemloading` become numeric `cpu_percent`/
`mem_percent`, `threadismonitor` becomes a real `is_monitored` boolean,
and `pid` becomes a real number (the agent reports it as a numeric
string, e.g. `"pid":"639"` - confirmed against a live server). Its
per-item field shape is inferred by analogy with the sibling
`get_software_info` action (same underlying process-monitor subsystem)
rather than verified against a populated list - see the
`MonitoredProcess` doc comment in `internal/DeviceOn-EIM/types.go`.


### Logs and operation records

```
eim-skill log system set-monitor <endpoint-or-name> <on|off>       # turn system log collection/reporting on or off

eim-skill log system list [--device-id <endpoint-or-name>] [--os <os>] [--keywords <substr>]
                           [--mode <mode>] [--level <level>] [--limit N] [--last <30m|12h|3d>]
                           [--begin <time>] [--end <time>]           # or an absolute range instead of --last
eim-skill log system delete-by-device <endpoint-or-name>

eim-skill log agent download <endpoint-or-name> [--output <path>]  # download the agent (client-side) log as a zip
                             [--content [--lines N]]           # or return the latest day's log tail directly

eim-skill log agent file <endpoint-or-name> <remote-path> [--lines N] [--output <path>]  # fetch one file by absolute path

eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
                               [--begin <time>] [--end <time>]       # or an absolute range instead of --last
eim-skill log operations delete-all
```

`--last` (on `log system list`, `log operations list`, and `device
monitor`) takes a relative time window instead of raw Unix-millisecond
timestamps - e.g. `--last 30m`, `--last 12h`, `--last 3d` - covering the
entries/samples from that far back through now; omit it for no time
filter. `--begin`/`--end` are an alternative, absolute way to specify the
same range - RFC3339 or `"YYYY-MM-DD HH:MM"` (e.g. `"2026-07-01 00:00"`);
either or both may be given (an omitted bound is unbounded), and they're
mutually exclusive with `--last`. `--limit` (default 20) simply caps how
many of the most recent entries come back - there's no `--page` to think
about, since this command always fetches from the start of the result
set.

`log agent download` is a three-step, undocumented (in AppHub-v2.3.0.md)
process confirmed against a live server via a real packet capture:
trigger a push (LwM2M write to resource 40001/0/27012, with a fresh
actionId and this server's own address), poll
`/api/devicemgr/logStatus?actionId=...` every 2 seconds up to a 3-minute
timeout until it reports "success", then fetch and save
`/api/devicemgr/agentLog/{endpoint}`. The result is a zip archive of
per-day log files plus the current day's log (default output filename:
`<endpoint>-agent-log.zip`) - this is a different log from `log system`
(the device's own systemd/OS log), it's the DeviceOn-EIM client process's
own log.

`--content` extracts the zip and returns the tail of the most recent
day's log entry directly as text (`data.content`) instead of requiring a
file - the smallest daily file observed live was still several MB, so
returning the whole thing isn't practical. "Most recent" prefers an entry
named exactly `EIM-Client.log` (the currently-active log, confirmed live
to duplicate that day's dated entry) when present; `--lines` (default
200) controls how many trailing lines come back. Without `--output`
given alongside `--content`, the zip itself is never written to disk -
pass both if you want the file too.

`log agent file <endpoint-or-name> <remote-path>` fetches one specific
file from a device's filesystem by absolute path (e.g.
`/media/recovery/EIM-Client/20260721.log`) - also undocumented, confirmed
against a live server via a real packet capture. It shares `log agent
download`'s trigger/poll/fetch shape, but against a different LwM2M
resource (40001/0/27023, which takes the file path directly) and a
different download endpoint
(`/api/devicemgr/log?filename=<endpoint>_<label>.zip&type=app&clientId=<endpoint>&value=<label>`,
not `/api/devicemgr/agentLog`) that requires a label tying the trigger to
the download (`appname` in the trigger, `value` in the download query) -
this reuses `actionId` itself for that label, since nothing in the
capture suggests it carries any other meaning. Unlike the agent log dump,
the returned zip always contains exactly one file (fetching a
nonexistent path fails outright at the trigger step with a clear server
error instead of returning something wrong), so its content is printed
directly by default; `--output` saves the raw file instead (e.g. for
binary content, or to also keep a copy), and `--lines` returns just the
trailing N lines instead of the whole file.

`log operations` (an audit log of admin actions) is not documented in
DeviceOn-EIM-v2.3.0.md - it was reverse-engineered from
`OperationRecordServlet.java`, and its field names/shapes (`orid`, `type`,
`event`, `target`, `username`, `ts`) are confirmed against a live server's
actual response (`data` is a plain JSON array, not the double-encoded
string the servlet's naming would suggest).

`log system set-monitor` is likewise undocumented (inferred from a
captured devicecontrol request using the `set_monitor_data_report`/
`systemd_log_monitor` action), but confirmed to succeed against a live
device (its effect is visible afterward in `log operations list`'s audit
trail). A device needs this turned on at least once before `log system
list` will return anything for it. `log system get-monitor` reports its
current state (`get_monitor_data_report` - the same action `device
get-monitor-report` uses for CPU/memory collection status, just reading
a different key from the same response).

`log system list` queries the system log entries a device has reported;
`delete-by-device` clears all of a device's entries at once (there is no
single-id delete or batch-delete - `delete-by-device` is the only removal
command). It's also undocumented in DeviceOn-EIM-v2.3.0.md -
reverse-engineered from a real packet capture, which confirmed the
endpoint path is `/api/systemLog/byPage` (capital `P`). An earlier version
of this CLI called the lowercase `/api/systemLog/bypage`, which the server
rejects with an HTTP 400 "Invalid path" error - that was a bug in this
client, not a genuine server limitation, and this whole command group had
been removed as a result before the mistake was found and fixed. `level`
and `pid` are normalized to real JSON numbers - the server reports both
as numeric strings (e.g. `"level":"4"`), confirmed against a live server.

### Sub-devices

```
eim-skill subdevice list [--parent <endpoint-or-name>]              # list sub-devices, optionally under one parent device
eim-skill subdevice get <device-id-or-name> [--service <name>] [--property <name>]  # services/properties + current real-time values ("twin")
eim-skill subdevice history <device-id-or-name> <service-name> <property-name>
  [--last <30m|12h|3d>] [--begin <time>] [--end <time>] [--count N]
eim-skill subdevice events <device-id-or-name> [--page N] [--limit N]
  [--last <30m|12h|3d>] [--begin <time>] [--end <time>]
```

Sub-devices ("子设备") are things registered under a parent
DeviceOn-EIM-managed device (the "edge") via a protocol like
Modbus/SNMP/executor/restful - each reports properties grouped into named
services, and can raise events. This is a separate, read-only query
surface backed by the `apphub-ithings` server (its own base URL,
`--ithings-base-url`/`ithings_base_url`, default `http://127.0.0.1:8082`)
- not documented in DeviceOn-EIM-v2.3.0.md for most of what it wraps
(only the sub-device list, twin, and history endpoints are; `events` is
undocumented entirely), confirmed against a live server via a real packet
capture. It accepts the same accesstoken as the main server's `/login` -
no separate ithings login.

`subdevice get` doubles as service/property discovery: there's no
separate schema-only endpoint wrapped here, so the "twin" (every
property ever reported, with its current value) is also how you find the
`service-name`/`property-name` pairs to pass to `subdevice history`.
`<device-id-or-name>` accepts either the sub-device's id or its name -
unlike parent devices' endpoint/devName, a sub-device's name isn't
guaranteed unique across different parents, so an ambiguous name errors
out listing the candidates rather than picking one.

`subdevice get`'s `--service`/`--property` narrow the twin down
client-side (the underlying endpoint has no filter for either, always
returning every reported property) - with both given, the single
matching property is returned directly instead of a list (or an error if
that exact service+property combination has no current value); with just
one of the two, every match comes back as a list.

`subdevice history`'s `--last`/`--begin`/`--end`/`--count` are sent to
match the documented request shape, but confirmed against a live server
to have **no observable filtering effect** - every range tested
(including deliberately absurd ones) returned the identical most-recent
~20 samples. `subdevice events`' time filters are untested (no sub-device
in the environment tested against had recorded events to verify with).

### OTA deployment

Each package type follows the same workflow: `list` queries the repo
server (m2m-management-edge) for packages available to push; `apply`/
`apply-group` (or `install`/`upgrade` for Windows) look up the requested
package by name+version in that listing, build its download path, and
push it to a device or a whole group via the same devicecontrol resource
the DeviceOn-EIM web console uses.

```
eim-skill ota linux list                                            # packages available in the repo
eim-skill ota linux installed <endpoint-or-name>                    # packages installed on a device
eim-skill ota linux status <endpoint-or-name>                       # most recent OTA task status
eim-skill ota linux apply <endpoint-or-name> <product-name> <version>
eim-skill ota linux apply-group <group-name> <product-name> <version> [--label <name>]

eim-skill ota android list
eim-skill ota android installed <endpoint-or-name>
eim-skill ota android status <endpoint-or-name>
eim-skill ota android apply <endpoint-or-name> <pkg-name> <version>
eim-skill ota android apply-group <group-name> <pkg-name> <version> [--label <name>]
eim-skill ota android uninstall <endpoint-or-name> <pkg-name>

eim-skill ota windows-exe list
eim-skill ota windows-exe installed <endpoint-or-name>
eim-skill ota windows-exe status <endpoint-or-name>
eim-skill ota windows-exe install <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe install-group <group-name> <project-name> <version>
eim-skill ota windows-exe upgrade <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe upgrade-group <group-name> <project-name> <version>
eim-skill ota windows-exe uninstall <endpoint-or-name> <project-name> <version> [--tool <tool>]

eim-skill ota windows-zip list
eim-skill ota windows-zip installed <endpoint-or-name>
eim-skill ota windows-zip status <endpoint-or-name>
eim-skill ota windows-zip install <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip install-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip update <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip update-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip uninstall <endpoint-or-name> <project-name> <version>

eim-skill ota docker list
eim-skill ota docker status <endpoint-or-name>
eim-skill ota docker apply <endpoint-or-name> <tag> <version>
eim-skill ota docker apply-group <group-name> <tag> <version> [--label <name>]
eim-skill ota docker uninstall <endpoint-or-name> <tag> <version>

eim-skill ota file list --type <type>
eim-skill ota file status <endpoint-or-name> --type <type>
eim-skill ota file deploy <endpoint-or-name> <file-name> <save-path> --type <type> [--exec]
eim-skill ota file deploy-group <group-name> <file-name> <save-path> --type <type> [--exec] [--label <name>]

eim-skill ota bsp status <endpoint-or-name>                         # BSP/system-image update status (status-only, no list/apply)
```

Notes:
- `windows-exe uninstall`'s `--tool` defaults to `"Inno Setup"`; pass
  `--tool "WiX Toolset"` explicitly if that's how the package was built.
  `install`/`upgrade` read it from the repo listing automatically, no flag
  needed either way.
- Android/Docker/Linux have one resource for both install and update (the
  server distinguishes by whether the package is already present); Windows
  has genuinely separate install vs. upgrade resources, so this CLI
  exposes them as separate commands rather than collapsing them.
- `ota file list`/`deploy`/`deploy-group` require `--type`, matching
  whatever category the file was uploaded under in the repo server (e.g.
  `deploy`) - there is no unconditional "list all files" endpoint. This
  `--type` is a freeform label, not an OS/platform name - `ota file list
  --type linux` (or `windows`/`android`) reliably returns an empty list
  (confirmed via a real bug report expecting it to return platform
  packages) unless a generic file happens to be uploaded under that
  literal category. For actual Linux/Android/Windows/Docker packages, use
  `ota linux list`/`ota android list`/`ota windows-exe list`/`ota
  windows-zip list`/`ota docker list` instead.
- `ota <type> status` (every type, including `bsp`) queries a device's
  most recent OTA/BSP task status. None of these are documented in
  AppHub-v2.3.0.md - found via an apifox-exported OpenAPI spec of AppHub's
  own web console, and confirmed to respond successfully against a live
  server, but that server had zero task history recorded, so no command's
  non-empty response shape is confirmed. `linux`/`android`/`windows-exe`/
  `windows-zip`/`docker` share a backend keyed by a "type" classifier
  that's only documented for Android (fixed value `"sg"`); all five reuse
  `"sg"` on an unverified assumption it's shared manager-wide. `bsp` has
  its own dedicated, type-param-free backend, so it carries less risk.
  See `skills/ota-deployment/SKILL.md` for the full breakdown.
- Verified live against a real server: all five repo-listing endpoints,
  and the full device-resolve → repo-lookup → error-handling pipeline
  (`ota file list --type deploy` returned real repo data matching this
  CLI's struct exactly). The final device-side deploy POST for each type
  is implemented from DeviceOn-EIM-v2.3.0.md's documented request/response
  shapes but wasn't exercised against a live device in this pass.

### Server configuration

```
eim-skill server get-config                                        # client-onboarding config blob
```

`server get-config` is not scoped to a device - it returns the server's
own client-onboarding config blob (`{"id":"WP.SRP","data":"<encrypted
blob>"}`). Paste `data` directly into a device agent's own configuration
UI to connect it to this server, instead of typing the server
address/credentials into the agent by hand - this mirrors the web
console's QR-code device-pairing feature (the same blob is what gets
encoded into the QR code shown there). Not documented in
DeviceOn-EIM-v2.3.0.md; confirmed against a live server.

## Configuration

`eim-skill` needs the DeviceOn-EIM server's base URL and login credentials,
plus the repo server's base URL for OTA commands and the ithings server's
base URL for `subdevice` commands. These are read from, in increasing
priority order:

1. `conf/config.yaml` (or the path given with `--config`):
   ```yaml
   base_url: http://127.0.0.1:30010
   username: admin
   password: admin
   repo_base_url: http://127.0.0.1:30001
   ithings_base_url: http://127.0.0.1:8082
   ```
2. Environment variables: `EIM_BASE_URL`, `EIM_USERNAME`, `EIM_PASSWORD`, `EIM_REPO_BASE_URL`, `EIM_ITHINGS_BASE_URL`
3. Flags: `--base-url`, `--username`, `--password`, `--repo-base-url`, `--ithings-base-url`

The repo and ithings servers both accept the same `accesstoken` issued by
the main server's `/login`, so no separate credentials are needed for
either.

### TLS trust (self-signed / internal CA servers)

If `base_url` (or the repo/ithings URLs) uses `https://` with a certificate
signed by an internal/private CA, requests fail with `x509: certificate
signed by unknown authority`. Fix with one of:

```yaml
# Recommended: trust one additional CA (PEM file), everything else still verified.
ca_cert_file: /path/to/internal-ca.pem

# Or: skip TLS verification entirely - only use this on a network path you
# already trust, since it removes protection against MITM attacks.
insecure_skip_verify: true
```

Also available as env vars (`EIM_CA_CERT_FILE`, `EIM_INSECURE_SKIP_VERIFY`)
or flags (`--ca-cert-file`, `--insecure`). `--insecure`/`insecure_skip_verify`
takes precedence over `--ca-cert-file`/`ca_cert_file` when both are set.

## MCP server

```
eim-skill mcp
```

Runs `eim-skill` as a [Model Context Protocol](https://modelcontextprotocol.io)
server over stdio, so an MCP-compatible agent/client can call its commands
directly as tools instead of shelling out to the CLI. Every runnable command
is auto-exposed as one MCP tool, named by its command path joined with `_`
(e.g. `device list` → `device_list`, `ota windows-exe apply-group` →
`ota_windows-exe_apply-group`). A tool's input schema is generated from its
command's positional args (parsed from `Use`, e.g. `<endpoint-or-name>` is
required, `[id...]` is an optional repeatable array) and its flags (typed
`string`/`boolean`/`integer`/`number` per the underlying pflag type, marked
required where the CLI does via `MarkFlagRequired`). Each tool call returns
the exact same JSON envelope as the equivalent CLI invocation, as its text
content, with `IsError` set to true when `status` is `500`.

To register it with an MCP client, point it at the built binary with `mcp`
as the argument, e.g.:

```json
{
  "mcpServers": {
    "eim-skill": {
      "command": "/path/to/eim-skill",
      "args": ["mcp"]
    }
  }
}
```

Configuration (base URL, credentials, repo server URL) is read the same way
as for direct CLI use (see [Configuration](#configuration) below) - set it
via `conf/config.yaml`, environment variables, or pass extra CLI flags in
`args` (e.g. `["mcp", "--config", "/path/to/config.yaml"]`).

## Skills

[`skills/`](skills/) has one `SKILL.md` per capability area (device
management, device monitoring, logs/records, OTA deployment,
sub-devices), each documenting when to use it and the exact commands -
written for an agent (or a person) deciding which part of this CLI to
reach for.

## How to build

```
$ make        ## build the eim-skill binary
$ make clean  ## clean the project
```

## Contributing

If you're interested in being a contributor, please see
[CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the
contribution workflow.
