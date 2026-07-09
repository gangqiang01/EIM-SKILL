# EIM-SKILL

`eim-skill` is a command-line skill tool that wraps
[DeviceLink](../../../MYPROJECT/BACKEND/DeviceLink)'s device management
REST API (documented in `/opt/AppHub-v2.3.0.md`) so device operations can be
driven from the command line or invoked as a skill by an agent.

## Supported commands

```
eim-skill device list                                        # list all devices
eim-skill device list --group <group-name>                    # list devices in a group
eim-skill device get <endpoint>                                # get device details
eim-skill device delete <endpoint>                             # delete a device
eim-skill device add-to-group <endpoint> <group>               # add a device to a group
eim-skill device monitor <endpoint> --type cpu|memory          # get cpu/memory monitoring data
  [--begin <unix-ms>] [--end <unix-ms>]
eim-skill device reboot <endpoint>                              # reboot a device now
eim-skill device shutdown <endpoint>                            # shut down a device now
eim-skill device set-volume <endpoint> <0-100>                  # set audio volume
eim-skill device set-backlight <endpoint> <0-100>                # set screen backlight
eim-skill group list                                             # list all device groups
```

`set-volume` and `set-backlight` are not documented in AppHub-v2.3.0.md;
they are implemented on the same devicecontrol resource as reboot/shutdown,
inferred from DeviceLink's `PluginsServlet` source. Verify against your
device firmware before relying on them in production.

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

## Configuration

`eim-skill` needs the DeviceLink server's base URL and login credentials,
plus the repo server's base URL. These are read from, in increasing
priority order:

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

## How to build

```
$ make        ## build the eim-skill binary
$ make clean  ## clean the project
```

## Contributing

If you're interested in being a contributor, please see
[CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the
contribution workflow.
