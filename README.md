# EIM-SKILL

`eim-skill` is a command-line skill tool that wraps
[DeviceLink](../../../MYPROJECT/BACKEND/DeviceLink)'s device management
REST API (documented in `/opt/AppHub-v2.3.0.md`) so device operations can be
driven from the command line or invoked as a skill by an agent.

## Supported commands

```
eim-skill device list                                        # list all devices
eim-skill device list --group <group-name>                    # list devices in a group
eim-skill device list --online|--offline                      # filter by online status
eim-skill device list --alarm                                  # devices with a software/hardware alarm
  [--group <group-name>]                                       # any of the above can combine with --group
eim-skill device get <endpoint>                                # get the full device record
eim-skill device details <endpoint>                            # get device system info (cpu/memory/storage/network)
eim-skill device delete <endpoint>                             # delete a device
eim-skill device add-to-group <endpoint> <group>               # add a device to a group
eim-skill device monitor <endpoint> --type cpu|memory          # get cpu/memory monitoring data
  [--begin <unix-ms>] [--end <unix-ms>]
eim-skill device reboot <endpoint>                              # reboot a device now
eim-skill device shutdown <endpoint>                            # shut down a device now
eim-skill device set-volume <endpoint> <0-100>                  # set audio volume
eim-skill device set-backlight <endpoint> <0-100>                # set screen backlight
eim-skill group list                                             # list all device groups
eim-skill group create <name> <type> [description] [--creator <username>]  # add a device group
```

`set-volume` and `set-backlight` are not documented in AppHub-v2.3.0.md;
they are implemented on the same devicecontrol resource as reboot/shutdown,
inferred from DeviceLink's `PluginsServlet` source. Verify against your
device firmware before relying on them in production.

`device details` is also undocumented (found in `ClientServlet`'s
`get_1_details`); it's a lighter-weight call than `device get` when you
only need the agent-reported system info blob (cpu, memory, storage,
network interfaces) rather than the full device record (online status,
group membership, etc).

### Monitoring and alerting

```
eim-skill device set-monitor <endpoint> <on|off>                 # overall hw monitoring on/off

eim-skill device threshold get <endpoint>                          # current threshold config
eim-skill device threshold set <endpoint> <metric> <on|off>        # configure one metric
  [--warn <n>] [--error <n>]
  # metric: cpu_temp, cpu_usage, memory_usage, storage_usage,
  #         battery_available, network_upload_speed, network_download_speed

eim-skill device process list <endpoint>                           # running processes + monitor status
eim-skill device process set-monitor <endpoint> <on|off>           # app/process monitoring on/off

eim-skill device docker info <endpoint>                            # deployed docker groups/containers
eim-skill device docker set-monitor <endpoint> <on|off>            # docker monitoring on/off

eim-skill device usb get-monitor <endpoint>                        # is USB change monitoring on?
eim-skill device usb set-monitor <endpoint> <on|off>               # USB change monitoring on/off
```

This mirrors the DeviceLink web console's "monitorAlert" feature: CPU,
memory, storage and network are monitored via configurable warn/error
thresholds (`device threshold`) rather than a live dashboard feed - pair
this with `device monitor --type cpu|memory` for historical CPU/memory
samples. Process and Docker monitoring here cover the on/off toggle only;
the web console's per-process/per-container selection isn't wired up yet.

### OTA (Linux packages)

```
eim-skill ota list-available                                    # packages available in the repo server
eim-skill ota list-installed <endpoint>                          # packages installed on a device
eim-skill ota apply <endpoint> <product-name> <version>          # install/update on one device
eim-skill ota apply-group <group-name> <product-name> <version>  # install/update on a whole group
  [--label <label-name>]
```

The OTA workflow mirrors the DeviceLink web console's Linux app-control
flow: `list-available` queries the repo server (m2m-management-edge) for
packages that can be pushed to devices, `list-installed` reads what's
already on a device, and `apply`/`apply-group` look up the requested
`<product-name>`/`<version>` in the repo listing and push it via the same
devicecontrol resource (`60001/0/28301`) the web console uses. This only
covers Linux devices/packages for now (Windows/Android/HarmonyOS/Docker use
different repo and devicecontrol resources and aren't wired up yet).

### Storage backend config (repo server)

```
eim-skill storage list
eim-skill storage add-s3 <platform> <endpoint-url> <bucket> <access-key> <secret-key>
eim-skill storage add-azure <container> <connection-string>
eim-skill storage activate <sid>
eim-skill storage delete <sid>
```

Configures the repo server's (m2m-management-edge) pluggable storage
backend used for OTA packages, app icons, etc. There is no raw-local-disk
mode; the bundled "local" deployment is actually a co-located MinIO
instance reachable via the same S3-compatible API as `add-s3`. `<platform>`
is stored verbatim as the backend's type - the server enforces no fixed
enum, so pick a consistent label (`"minio"`, `"s3"`, `"oss"`). `storage
list` never returns the secret key.

### Data forwarding (AMQP/MQTT/Kafka)

```
eim-skill dataforward list
eim-skill dataforward get <name>
eim-skill dataforward create <name> [description]
eim-skill dataforward set <name> <amqp|kafka|mqtt> [description]
  --model-id <id> --host <host:port> --username <u> --password <p>
  --topic <t>            # mqtt/kafka
  --queue <q>             # amqp
  --client-id <id> --qos <0-2> --retain    # mqtt only
eim-skill dataforward enable <name>
eim-skill dataforward disable <name>
eim-skill dataforward delete <id>
eim-skill dataforward logs
```

Pushes sub-device telemetry (selected by device model) to an external
broker. This is served by a separate microservice called `ithings`
(port 8082 by convention), not the Java DeviceLink server - see
`ithings_base_url` below. `create` alone makes an empty, disabled rule;
`set` configures its source/destination.

### Sub-devices (Modbus/OPC UA/IEC104/S7/SNMP)

```
eim-skill subdevice model list
eim-skill subdevice model get <model-name>
eim-skill subdevice model import-excel <file.xlsx>

eim-skill subdevice list
eim-skill subdevice get <device-id>
eim-skill subdevice import-excel <file.xlsx> [--edge-ids a,b | --group <name>]
eim-skill subdevice set-collection <device-id> <on|off>
eim-skill subdevice twin <device-id>
eim-skill subdevice twin-history <device-id> <service-name> <property-name> [--begin <unix-ms>] [--end <unix-ms>]
eim-skill subdevice protocols
```

Also served by the `ithings` microservice. A gateway device polls
non-IP sub-devices (PLCs, sensors, meters) over an industrial protocol on
their behalf. `subdevice twin` returns a sub-device's real-time collected
data (its "device shadow"); `twin-history` returns a property's samples
over a time range. Models and sub-devices are only created via Excel
upload - there's no one-at-a-time create command.

Excel workbook layout - `subdevice model import-excel`:
- `deviceModel`: name, description, manufacturer, industry, dataFormat (one row)
- `property`: module, name, description, dataType, maxValue, minValue, writeAble, unit
- `event`: module, name, description

Excel workbook layout - `subdevice import-excel`:
- `subDevice`: deviceId, name, edgeId, description, deviceType, liftTimeOfDesiredValue, deviceModel, protocolType, protocol, tags, reliable, reportOnChange (one template row)
- `propertyConfig`, `eventConfig`, `commandConfig`: module, name, plus protocol-specific access-config columns

Observed server-supported protocol types are `"modbus"` and `"snmp"` (run
`subdevice protocols` to see what your deployment actually has); `"opcua"`,
`"iec104"`, and `"s7"` were not found registered in any deployment
researched for this CLI - if your server has them, this command will show
it, and the `protocolType`/`protocol` columns in the sub-device Excel
template should work the same way. `subdevice import-excel` runs
asynchronously server-side - a successful response means the upload was
accepted, not that every row is persisted yet.

### Logs and operation records

```
eim-skill log system list [--page N] [--limit N] [--device-id <ep>] [--os <os>] [--keywords <kw>] [--begin <unix-ms>] [--end <unix-ms>]
eim-skill log system delete <id>
eim-skill log system delete-by-device <device-id>
eim-skill log system batch-delete <id> [id...]

eim-skill log operations list [--page N] [--limit N] [--target <substr>] [--begin <unix-ms>] [--end <unix-ms>]
eim-skill log operations delete <id>
eim-skill log operations delete-all
eim-skill log operations batch-delete <id> [id...]

eim-skill log agent request <endpoint> <callback-url> <action-id>
eim-skill log agent status <action-id>
eim-skill log agent download <endpoint> [--out <file>]
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

## Configuration

`eim-skill` needs the DeviceLink server's base URL and login credentials,
plus the repo and ithings servers' base URLs. These are read from, in
increasing priority order:

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

## Skills

[`skills/`](skills/) has one `SKILL.md` per capability area (device
management, device monitoring, OTA, storage config, data forwarding,
sub-devices, logs/records), each documenting when to use it and the exact
commands - written for an agent (or a person) deciding which part of this
CLI to reach for.

## How to build

```
$ make        ## build the eim-skill binary
$ make clean  ## clean the project
```

## Contributing

If you're interested in being a contributor, please see
[CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the
contribution workflow.
