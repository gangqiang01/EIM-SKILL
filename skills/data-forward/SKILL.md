---
name: data-forward
description: Configure DeviceLink's data-forward rules that push sub-device telemetry to an external AMQP, MQTT, or Kafka broker, and inspect forwarding delivery logs. Use this whenever the user asks to forward, bridge, or export device data to a message queue/broker via the eim-skill CLI.
---

# Data forwarding (AMQP/MQTT/Kafka)

Wraps the `ithings` microservice's data-forward rules (push sub-device
telemetry, selected by device model, to an external broker). Requires
`eim-skill` built and configured (see repo README) - talks to the
`ithings` server, reusing the main AppHub login token.

## When to use this skill

- "What data-forward rules exist?" → `dataforward list` / `dataforward get`
- "Forward data from device model M to our AMQP/MQTT/Kafka broker" → `dataforward create` then `dataforward set`
- "Turn forwarding rule R on/off" → `dataforward enable` / `dataforward disable`
- "Remove forwarding rule R" → `dataforward delete`
- "Did forwarding rule R actually deliver anything?" → `dataforward logs`

## Commands

```
eim-skill dataforward list
eim-skill dataforward get <name>
eim-skill dataforward create <name> [description]
eim-skill dataforward set <name> <amqp|kafka|mqtt> [description]
  --model-id <id>                  # forward every device of this device-model (source)
  --host <host:port> --username <u> --password <p>
  --topic <t>            # mqtt/kafka
  --queue <q>            # amqp
  --client-id <id> --qos <0-2> --retain    # mqtt only
eim-skill dataforward enable <name>
eim-skill dataforward disable <name>
eim-skill dataforward delete <id>          # id, not name - see "dataforward list"
eim-skill dataforward logs
```

A rule must be created first (`create`), then configured (`set`) before
enabling it - `create` alone makes an empty, disabled rule.

## Notes and caveats

- Sources are selected by device model only in this CLI (`--model-id`);
  the server also supports per-edge/per-tag sources, not exposed here.
- List/get/logs commands print the server's raw JSON response verbatim
  (not reshaped into a fixed schema) - the exact field names come straight
  from the `ithings` server.
