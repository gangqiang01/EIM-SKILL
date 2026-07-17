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
eim-skill device reboot <endpoint-or-name>                         # reboot a device now
eim-skill device shutdown <endpoint-or-name>                       # shut down a device now
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
report.

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
group membership, etc).

`device counts`/`device health`/`group status` are documented in
DeviceOn-EIM-v2.3.0.md but return a flat `[{"name":...,"value":...}]` array
(or, for `group status`, `"gname"` instead of `"name"`); this CLI
normalizes them into a single object with clear field names.

### Monitoring and alerting

```
eim-skill device get-monitor <endpoint-or-name>                      # cpu/memory/storage/battery/network snapshot

eim-skill device threshold get <endpoint-or-name>                   # current threshold config
eim-skill device threshold set <endpoint-or-name> <metric> <on|off> # configure one metric
  [--warn <n>] [--error <n>]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process monitored <endpoint-or-name>               # processes currently selected for monitoring

eim-skill device docker monitored <endpoint-or-name>                 # docker containers currently selected for monitoring

eim-skill device usb get-monitor <endpoint-or-name>                 # is USB storage device access enabled?
eim-skill device usb set-monitor <endpoint-or-name> <on|off>        # USB storage device access on/off
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

There is no `device set-monitor` (overall hardware monitoring on/off) -
it used the `set_isMonitor` devicecontrol action and has been removed;
`device threshold set` already lets you disable alerting per-metric.
There is also no `device process set-monitor`/`device docker
set-monitor`: they used the `set_isProcessMonitor`/
`set_isDockerMonitor` devicecontrol actions and were confirmed against a
live server to return success without actually changing anything, so
they were removed rather than shipped non-functional.

There is also no `device process list` (every process regardless of
monitor status) - only `device process monitored` remains, which already
covers the practical use case. `device docker monitored` is now the Docker
analogue of `device process monitored`: it returns only the containers
currently selected for monitoring (`ismonitor == true`), not every
deployed container. An earlier version instead read from an undocumented
LwM2M resource (`.../60001/0/27300`) assuming a response grouped by
compose project - that returned an empty result on a live server with
real containers deployed, so it was switched to the working
devicecontrol `get_docker_info` action (confirmed via a real packet
capture) and filtered to the monitored subset client-side, matching
`device process monitored`'s shape/intent.

`device usb get-monitor`/`set-monitor` control whether USB storage
devices (e.g. a USB flash drive) can be used on the device - not USB
connect/disconnect alerting, as an earlier version assumed. They use the
`get_usb_storage_status`/`set_usb_storage_status` devicecontrol actions,
confirmed against a live server via a real packet capture; `set-monitor`
was additionally cross-checked in `log operations list`'s audit trail,
where it's logged under the friendly event names `enableUsb`/
`disableUsb` (most actions are logged as raw JSON instead), confirming
the server specially recognizes it as a real, meaningful action. An
earlier version instead used `get_usb_change`/`set_peripheral_monitor`
(alert on USB connect/disconnect) - that `set_peripheral_monitor` calls
were logged as raw unrecognized JSON in the same audit trail was a strong
signal it wasn't a real action either, so it was replaced. Some devices
return an empty result for `get_usb_storage_status` rather than a
definite value; this CLI normalizes that to `enabled: false`.

`device monitor`'s `value` is the usage percentage (0-100) for whichever
`--type` you asked for - CPU usage % for `--type cpu`, memory usage % for
`--type memory` (inferred from observed values matching
`device get-monitor`'s `cpu.usage_percent`/`memory.usage_percent`, not a
spec). It's parsed leniently (JSON number or numeric string, both
normalized) since some servers report it as a string.

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
`mem_percent`, `threadismonitor` becomes a real `is_monitored` boolean.
Its per-item field shape is inferred by analogy with the sibling
`get_software_info` action (same underlying process-monitor subsystem)
rather than verified against a populated list - see the
`MonitoredProcess` doc comment in `internal/DeviceOn-EIM/types.go`.


### Logs and operation records

```
eim-skill log system set-monitor <endpoint-or-name> <on|off>       # turn system log collection/reporting on or off

eim-skill log system list [--device-id <endpoint-or-name>] [--os <os>] [--keywords <substr>]
                           [--mode <mode>] [--level <level>] [--limit N] [--last <30m|12h|3d>]
eim-skill log system delete-by-device <endpoint-or-name>

eim-skill log operations list [--limit N] [--target <substr>] [--last <30m|12h|3d>]
eim-skill log operations delete-all
```

`--last` (on `log system list`, `log operations list`, and `device
monitor`) takes a relative time window instead of raw Unix-millisecond
timestamps - e.g. `--last 30m`, `--last 12h`, `--last 3d` - covering the
entries/samples from that far back through now; omit it for no time
filter. `--limit` (default 20) simply caps how many of the most recent
entries come back - there's no `--page` to think about, since this
command always fetches from the start of the result set.

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
list` will return anything for it.

`log system list` queries the system log entries a device has reported;
`delete-by-device` clears all of a device's entries at once (there is no
single-id delete or batch-delete - `delete-by-device` is the only removal
command). It's also undocumented in DeviceOn-EIM-v2.3.0.md -
reverse-engineered from a real packet capture, which confirmed the
endpoint path is `/api/systemLog/byPage` (capital `P`). An earlier version
of this CLI called the lowercase `/api/systemLog/bypage`, which the server
rejects with an HTTP 400 "Invalid path" error - that was a bug in this
client, not a genuine server limitation, and this whole command group had
been removed as a result before the mistake was found and fixed.

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
  `deploy`) - there is no unconditional "list all files" endpoint.
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

## Configuration

`eim-skill` needs the DeviceOn-EIM server's base URL and login credentials,
plus the repo server's base URL for OTA commands. These are read from, in
increasing priority order:

1. `conf/config.yaml` (or the path given with `--config`):
   ```yaml
   base_url: http://127.0.0.1:30010
   username: admin
   password: admin
   repo_base_url: http://127.0.0.1:30001
   ```
2. Environment variables: `EIM_BASE_URL`, `EIM_USERNAME`, `EIM_PASSWORD`, `EIM_REPO_BASE_URL`
3. Flags: `--base-url`, `--username`, `--password`, `--repo-base-url`

The repo server accepts the same `accesstoken` issued by the main server's
`/login`, so no separate repo-server credentials are needed.

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
management, device monitoring, logs/records, OTA deployment), each
documenting when to use it and the exact commands - written for an agent
(or a person) deciding which part of this CLI to reach for.

## How to build

```
$ make        ## build the eim-skill binary
$ make clean  ## clean the project
```

## Contributing

If you're interested in being a contributor, please see
[CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the
contribution workflow.
