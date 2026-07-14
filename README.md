# EIM-SKILL

`eim-skill` is a command-line skill tool that wraps
[DeviceOn-EIM](../../../MYPROJECT/BACKEND/DeviceLink)'s device management
REST API (documented in `/opt/AppHub-v2.3.0.md`) so device operations can be
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
  [--begin <unix-ms>] [--end <unix-ms>]
eim-skill device reboot <endpoint-or-name>                         # reboot a device now
eim-skill device shutdown <endpoint-or-name>                       # shut down a device now
eim-skill device set-volume <endpoint-or-name> <0-100>             # set audio volume
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

`set-volume` and `set-backlight` are not documented in AppHub-v2.3.0.md;
they are implemented on the same devicecontrol resource as reboot/shutdown,
inferred from DeviceLink's `PluginsServlet` source. Verify against your
device firmware before relying on them in production.

`reboot-group`/`shutdown-group`/`set-volume-group`/`set-backlight-group`
target every device in a group with a single call, using AppHub's documented
group-batch devicecontrol resource (`PUT
/api/plugins/batch/{groupname}/40008/0/27601`) rather than looping over each
device individually. `--label` optionally narrows the target to devices
carrying that label.

`device details` is also undocumented (found in `ClientServlet`'s
`get_1_details`); it's a lighter-weight call than `device get` when you
only need the agent-reported system info blob (cpu, memory, storage,
network interfaces) rather than the full device record (online status,
group membership, etc).

`device counts`/`device health`/`group status` are documented in
AppHub-v2.3.0.md but return a flat `[{"name":...,"value":...}]` array
(or, for `group status`, `"gname"` instead of `"name"`); this CLI
normalizes them into a single object with clear field names.

### Monitoring and alerting

```
eim-skill device get-monitor <endpoint-or-name>                      # cpu/memory/storage/battery/network snapshot

eim-skill device set-monitor <endpoint-or-name> <on|off>            # overall hw monitoring on/off

eim-skill device threshold get <endpoint-or-name>                   # current threshold config
eim-skill device threshold set <endpoint-or-name> <metric> <on|off> # configure one metric
  [--warn <n>] [--error <n>]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process list <endpoint-or-name>                    # running processes + monitor status
eim-skill device process monitored <endpoint-or-name>               # processes currently selected for monitoring
eim-skill device process set-monitor <endpoint-or-name> <on|off>    # app/process monitoring on/off

eim-skill device docker info <endpoint-or-name>                     # deployed docker groups/containers
eim-skill device docker set-monitor <endpoint-or-name> <on|off>     # docker monitoring on/off

eim-skill device usb get-monitor <endpoint-or-name>                 # is USB change monitoring on?
eim-skill device usb set-monitor <endpoint-or-name> <on|off>        # USB change monitoring on/off
```

This mirrors the DeviceLink web console's "monitorAlert" feature: CPU,
memory, storage and network are monitored via configurable warn/error
thresholds (`device threshold`) rather than a live dashboard feed - pair
this with `device monitor --type cpu|memory` for historical CPU/memory
samples, or `device get-monitor` for a live snapshot (see below). Process
and Docker monitoring here cover the on/off toggle only; the web
console's per-process/per-container selection isn't wired up yet.

`device get-monitor` is not documented in AppHub-v2.3.0.md (found by
inspecting real traffic to the `get_board_info` devicecontrol action). Its
response is normalized: the raw agent payload mixes numbers, numeric
strings, and `"N/A"` placeholders inconsistently, so this CLI parses it
into consistent `float64` fields (`null` when unavailable) with
self-documenting names (`usage_percent`, `total_gb`, etc). Units/semantics
are inferred from observed values, not a spec - see the `BoardInfo`
doc comment in `internal/apphub/types.go`.

`device process monitored` is also undocumented (the `get_monitor_app`
devicecontrol action, found the same way) and normalized similarly -
`threadcpuloading`/`threadmemloading` become numeric `cpu_percent`/
`mem_percent`, `threadismonitor` becomes a real `is_monitored` boolean.
Its per-item field shape is inferred by analogy with the confirmed
`device process list` shape (same underlying subsystem) rather than
verified against a populated list - see the `MonitoredProcess` doc
comment in `internal/apphub/types.go`.

### Logs and operation records

```
eim-skill log system list [--page N] [--limit N] [--device-id <endpoint-or-name>] [--os <os>] [--keywords <kw>] [--begin <unix-ms>] [--end <unix-ms>]
eim-skill log system delete <id>
eim-skill log system delete-by-device <endpoint-or-name>
eim-skill log system batch-delete <id> [id...]

eim-skill log operations list [--page N] [--limit N] [--target <substr>] [--begin <unix-ms>] [--end <unix-ms>]
eim-skill log operations delete <id>
eim-skill log operations delete-all
eim-skill log operations batch-delete <id> [id...]

eim-skill log agent request <endpoint-or-name> <callback-url> <action-id>
eim-skill log agent status <action-id>
eim-skill log agent download <endpoint-or-name> [--out <file>]
```

`log operations` (an audit log of admin actions) is not documented in
AppHub-v2.3.0.md - see `OperationRecordServlet.java`. Agent/client log
retrieval is asynchronous and needs a reachable callback server: the
AppHub server can't push a file to you directly, so `log agent request`
asks the device's agent to upload its log to `<callback-url>` (which must
be an HTTP address the device can reach, running a server that accepts the
upload - this CLI does not provide one). Poll `log agent status` until it
reports `"success"`, then `log agent download` (works only once - the
server deletes the file after serving it).

### OTA deployment

Each package type follows the same workflow: `list` queries the repo
server (m2m-management-edge) for packages available to push; `apply`/
`apply-group` (or `install`/`upgrade` for Windows) look up the requested
package by name+version in that listing, build its download path, and
push it to a device or a whole group via the same devicecontrol resource
the DeviceLink web console uses.

```
eim-skill ota linux list                                            # packages available in the repo
eim-skill ota linux installed <endpoint-or-name>                    # packages installed on a device
eim-skill ota linux apply <endpoint-or-name> <product-name> <version>
eim-skill ota linux apply-group <group-name> <product-name> <version> [--label <name>]

eim-skill ota android list
eim-skill ota android installed <endpoint-or-name>
eim-skill ota android apply <endpoint-or-name> <pkg-name> <version>
eim-skill ota android apply-group <group-name> <pkg-name> <version> [--label <name>]
eim-skill ota android uninstall <endpoint-or-name> <pkg-name>

eim-skill ota windows-exe list
eim-skill ota windows-exe installed <endpoint-or-name>
eim-skill ota windows-exe install <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe install-group <group-name> <project-name> <version>
eim-skill ota windows-exe upgrade <endpoint-or-name> <project-name> <version>
eim-skill ota windows-exe upgrade-group <group-name> <project-name> <version>
eim-skill ota windows-exe uninstall <endpoint-or-name> <project-name> <version> --tool <tool>

eim-skill ota windows-zip list
eim-skill ota windows-zip installed <endpoint-or-name>
eim-skill ota windows-zip install <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip install-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip update <endpoint-or-name> <project-name> <version> [--parameter <args>]
eim-skill ota windows-zip update-group <group-name> <project-name> <version> [--parameter <args>] [--label <name>]
eim-skill ota windows-zip uninstall <endpoint-or-name> <project-name> <version>

eim-skill ota docker list
eim-skill ota docker apply <endpoint-or-name> <tag> <version>
eim-skill ota docker apply-group <group-name> <tag> <version> [--label <name>]
eim-skill ota docker uninstall <endpoint-or-name> <tag> <version>

eim-skill ota file list --type <type>
eim-skill ota file deploy <endpoint-or-name> <file-name> <save-path> --type <type> [--exec]
eim-skill ota file deploy-group <group-name> <file-name> <save-path> --type <type> [--exec] [--label <name>]
```

Notes:
- `windows-exe` needs `--tool` matching how the package was uploaded
  ("Inno Setup" or "WiX Toolset") for uninstall; `install`/`upgrade` read
  it from the repo listing automatically.
- Android/Docker/Linux have one resource for both install and update (the
  server distinguishes by whether the package is already present); Windows
  has genuinely separate install vs. upgrade resources, so this CLI
  exposes them as separate commands rather than collapsing them.
- `ota file list`/`deploy`/`deploy-group` require `--type`, matching
  whatever category the file was uploaded under in the repo server (e.g.
  `deploy`) - there is no unconditional "list all files" endpoint.
- Verified live against a real server: all five repo-listing endpoints,
  and the full device-resolve → repo-lookup → error-handling pipeline
  (`ota file list --type deploy` returned real repo data matching this
  CLI's struct exactly). The final device-side deploy POST for each type
  is implemented from AppHub-v2.3.0.md's documented request/response
  shapes but wasn't exercised against a live device in this pass.

## Configuration

`eim-skill` needs the DeviceLink server's base URL and login credentials,
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
