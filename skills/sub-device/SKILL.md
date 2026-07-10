---
name: sub-device
description: Manage sub-devices connected to a gateway over an industrial protocol (Modbus, SNMP, and others) - define sub-device models and sub-devices by uploading Excel workbooks, start/stop data collection, and read a sub-device's real-time or historical collected data. Use this whenever the user asks about Modbus/OPC UA/IEC104/Siemens S7/SNMP device configuration or sub-device telemetry via the eim-skill CLI.
---

# Sub-device management (Modbus/OPC UA/IEC104/S7/SNMP)

Wraps the `ithings` microservice's sub-device feature: a gateway device
polls non-IP devices (PLCs, sensors, meters) on their behalf over an
industrial protocol. Requires `eim-skill` built and configured (see repo
README) - talks to the `ithings` server, reusing the main AppHub login
token.

## When to use this skill

- "What sub-device models/schemas exist?" → `subdevice model list` / `subdevice model get`
- "Define a new sub-device model/schema" → `subdevice model import-excel`
- "What sub-devices are configured on gateway X?" → `subdevice list` / `subdevice get`
- "Add a Modbus/SNMP sub-device to gateway X" → `subdevice import-excel`
- "Start/stop collecting data from sub-device D" → `subdevice set-collection`
- "What's sub-device D reporting right now?" → `subdevice twin`
- "What did sub-device D's property P read between time A and B?" → `subdevice twin-history`
- "What protocols does this server support?" → `subdevice protocols`

## Commands

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

**Models and sub-devices are only created via Excel upload - there is no
one-at-a-time create command.** `<edge-id>` (used in `--edge-ids`) is the
gateway device's endpoint, the same id `device list` shows.

### Excel workbook layout

`subdevice model import-excel` expects sheets:
- `deviceModel`: name, description, manufacturer, industry, dataFormat (one data row)
- `property`: module, name, description, dataType, maxValue, minValue, writeAble, unit
- `event`: module, name, description

`subdevice import-excel` expects sheets:
- `subDevice`: deviceId, name, edgeId, description, deviceType, liftTimeOfDesiredValue, deviceModel, protocolType, protocol, tags, reliable, reportOnChange (one data row - a template applied to every resolved gateway)
- `propertyConfig`, `eventConfig`, `commandConfig`: module, name, plus protocol-specific access-config columns

`protocolType`/`protocol` in the `subDevice` sheet select the connection
protocol and its connection config (a JSON string), e.g. for Modbus TCP:
`protocolType=modbus`, `protocol={"mtype":"tcp","tprot":{"ip_addr":"172.21.73.85","port":"502"}}`.

## Notes and caveats

- Observed server-supported protocol types are "modbus" and "snmp" (check
  `subdevice protocols` for what your deployment actually has registered)
  - "opcua", "iec104", and "s7" were not found registered in any
    deployment researched so far. If your server has them configured,
    `subdevice protocols` will show it, and the workbook's
    `protocolType`/`protocol` columns should work the same way once you
    know that protocol's connection-JSON shape; if not, they need to be
    added server-side first.
- `subdevice import-excel` runs asynchronously server-side (a background
  job) - the command returning success means the upload was accepted, not
  that every row has been persisted yet. There's no job-status endpoint to
  poll; check `subdevice list` after a short wait to confirm.
- List/get/twin/protocols commands print the server's raw JSON response
  verbatim (not reshaped into a fixed schema).
