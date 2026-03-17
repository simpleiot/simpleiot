# IoT Data Models: Points vs Structured Payloads

- Author: Cliff Brake, last updated: 2026-03-10
- Status: discussion

## Problem

SIOT uses a flat point model where each point has a single value (float, string,
or bytes). This works well for independent values that change at different rates
and need to be merged/synced independently. However, some measurements are
logically compound — for example, a power quality reading with average, peak,
and RMS values that are computed together and meaningless if separated.

The question is: how should SIOT handle compound/structured data while
preserving the simplicity and mergeability of the point model?

## How Other Systems Handle This

### Golioth (LightDB Stream)

Golioth takes a **document-oriented** approach. Devices publish arbitrary nested
JSON (or CBOR) payloads to LightDB Stream:

```json
{
	"temp": { "average": 22.5, "peak": 28.1 },
	"humidity": 65.2,
	"ts": 1719592181
}
```

Key characteristics:

- **No fixed schema** — the device defines the structure, and it can change on
  the fly.
- **Atomic writes** — all fields in a message are stored together.
- **State vs Stream separation** — LightDB State holds last-known values
  (read/write), while LightDB Stream is append-only time series.
- **CBOR support** — for constrained devices needing smaller payloads.
- **Pipelines** — server-side transformation and routing to external databases
  (InfluxDB, MongoDB Time Series, etc.).
- **No built-in merge/sync** — Golioth is cloud-first. Devices push data up;
  there is no bidirectional sync of state between peers.

**Tradeoff**: Maximum flexibility and simplicity for developers, but no support
for distributed merge or independent field updates. If two sources modify
different fields of the same document, there is no automatic conflict
resolution.

Reference:
[Golioth LightDB Stream](https://docs.golioth.io/cloud/services/lightdb-stream),
[Sending data by object](https://docs.golioth.io/firmware/golioth-firmware-sdk/light-db-stream/by-object/)

### AWS IoT / Azure IoT

Both major cloud platforms use **JSON telemetry messages** over MQTT:

```json
{
	"deviceId": "sensor-01",
	"temperature": 22.5,
	"acceleration": { "x": 0.1, "y": -0.3, "z": 9.8 },
	"timestamp": "2026-03-10T12:00:00Z"
}
```

Key characteristics:

- **Free-form JSON** — the device defines the payload structure.
- **Cloud-side processing** — rules engines, event processors, and stream
  analytics handle transformation and routing.
- **Device twins/shadows** — a separate mechanism for desired vs reported state,
  stored as JSON documents.
- **No edge-to-edge sync** — these are hub-and-spoke architectures. Data flows
  device → cloud. Configuration flows cloud → device.

**Tradeoff**: Very flexible, well-tooled cloud ecosystem, but no peer-to-peer
synchronization. The device twin model handles state but not time-series
history.

### SenML (RFC 8428)

SenML (Sensor Measurement Lists) is an IETF standard that is the closest to
SIOT's point model. It defines an array of individual measurements:

```json
[
	{
		"bn": "urn:dev:ow:10e2073a01080063:",
		"n": "voltage",
		"u": "V",
		"v": 120.1
	},
	{ "n": "current", "u": "A", "v": 1.2 },
	{ "n": "power", "u": "W", "v": 144.12 }
]
```

Key characteristics:

- **One value per record** — each SenML record contains exactly one of: numeric
  value (`v`), string value (`vs`), boolean value (`vb`), or data value (`vd`).
- **Base fields for compression** — base name (`bn`), base time (`bt`), base
  unit (`bu`) reduce repetition across records in a pack.
- **Multiple encodings** — JSON, CBOR, XML, and EXI are all defined.
- **Units and metadata** — built-in support for units and update time.
- **Flat by design** — explicitly does not support nested or compound values.
  Multiple related measurements from the same sensor are sent as separate
  records in the same SenML Pack.

**Handling compound data**: SenML handles the average/peak scenario by sending
two records in the same pack with different names (e.g., `voltage_avg` and
`voltage_peak`). They share a base name and base time but are separate records.
The pack provides a grouping mechanism, but there is no formal way to declare
that two records are semantically coupled.

**Tradeoff**: Standardized, well-defined, efficient for constrained devices. But
limited to flat, single-value measurements. No mechanism for expressing that
certain values form an atomic group. No built-in sync or merge semantics.

Reference: [RFC 8428](https://www.rfc-editor.org/rfc/rfc8428.html),
[SenML overview](https://umair-iftikhar.medium.com/senml-compact-json-for-sensor-data-with-metadata-2698efc7a915)

### Sparkplug B

Sparkplug B is an Eclipse Foundation specification built on MQTT, primarily
targeting industrial IoT (SCADA, PLC, factory automation):

```
Topic: spBv1.0/GroupA/DDATA/NodeA/DeviceA
Payload (Protobuf):
  timestamp: 1710072000000
  seq: 42
  metrics:
    - name: "Motor/Current"
      alias: 1
      timestamp: 1710072000000
      datatype: Float
      value: 12.5
    - name: "Motor/Voltage"
      alias: 2
      timestamp: 1710072000000
      datatype: Float
      value: 480.2
```

Key characteristics:

- **Protobuf encoding** — binary, efficient, schema-defined payloads.
- **Metrics array** — each message contains an array of metrics (name, value,
  timestamp, datatype). Similar to SenML but with richer metadata.
- **Birth/Death certificates** — devices publish DBIRTH messages declaring all
  their metrics on connect, and DDEATH on disconnect. This provides schema
  discovery.
- **Metric aliasing** — after the birth certificate, metrics can be referenced
  by numeric alias instead of string name, reducing bandwidth.
- **Templates** — support for user-defined compound types (UDTs). A template
  defines a group of metrics that belong together, similar to a struct.
- **Datasets** — tabular data type with rows, columns, and mixed types.
- **State management** — the broker maintains awareness of device state through
  birth/death certificates and session management.
- **Store-and-forward** — designed for intermittent connectivity with buffering.

**Handling compound data**: Sparkplug B has two mechanisms:

1. **Templates (UDTs)**: Define a named type with multiple metrics. A
   "PowerQuality" template could contain `average`, `peak`, and `rms` as member
   metrics. Templates are declared in the birth certificate and can be nested.
2. **Hierarchical naming**: Use `/` separators in metric names (e.g.,
   `Motor/Current`, `Motor/Voltage`) to logically group related metrics.

**Tradeoff**: Rich, well-specified, good for industrial use cases. But complex —
the specification is large and Protobuf adds implementation burden on
constrained devices. No built-in distributed sync (it assumes a central SCADA
host).

Reference:
[Sparkplug Specification](https://sparkplug.eclipse.org/specification/version/2.2/documents/sparkplug-specification-2.2.pdf),
[HiveMQ Sparkplug Payload Structures](https://www.hivemq.com/blog/mqtt-payload-structures-iiot/)

## Comparison

| Feature                     | SIOT Points          | Golioth              | AWS/Azure            | SenML                 | Sparkplug B        |
| --------------------------- | -------------------- | -------------------- | -------------------- | --------------------- | ------------------ |
| Data unit                   | Single point         | JSON document        | JSON document        | Single record         | Metric             |
| Compound data               | No (separate points) | Native (nested JSON) | Native (nested JSON) | No (separate records) | Templates/UDTs     |
| Encoding                    | Protobuf/binary      | JSON/CBOR            | JSON                 | JSON/CBOR/XML         | Protobuf           |
| Schema                      | Implicit (node type) | None (free-form)     | None (free-form)     | Minimal (units)       | Birth certificates |
| Atomic multi-field update   | No                   | Yes                  | Yes                  | Pack (loose)          | Template instance  |
| Independent field merge     | Yes                  | No                   | No                   | N/A                   | No                 |
| Bidirectional sync          | Yes                  | No                   | No                   | N/A                   | No                 |
| Constrained device friendly | Yes                  | Medium (CBOR)        | No (JSON)            | Yes (CBOR)            | Medium (Protobuf)  |

## Key Observation

Most IoT platforms don't face the compound-data tension because they use
document-oriented messages and don't need to merge individual fields. SIOT's
point model exists specifically to enable **bidirectional synchronization with
independent field merging** — a feature none of the other platforms support.

The fundamental tradeoff is:

- **Document model** (Golioth, AWS, Azure): Easy compound data, no merge
  capability.
- **Flat measurement model** (SenML, SIOT points): Great mergeability, awkward
  compound data.
- **Hybrid model** (Sparkplug templates): Supports compound types but adds
  significant complexity and still lacks distributed merge.

## Proposal for SIOT

A pragmatic hybrid approach that preserves the point model's strengths:

1. **Keep flat points as the default** for values that change independently
   (temperature, relay state, configuration settings). This preserves
   mergeability and per-field subscriptions via NATS subjects.

2. **Allow compound points** using the JSON or CBOR data type for logically
   coupled values. A point with `type=powerQuality` and data type=JSON
   containing `{"average": 22.5, "peak": 28.1, "rms": 23.0}` is still a single
   point in the stream — syncable and mergeable at the point level.

3. **Mergeability is preserved at the right granularity**. If average and peak
   are computed from the same measurement window, they should merge as a unit.
   You would never want to merge an average from one source with a peak from
   another. The merge boundary should match the semantic boundary.

4. **Node type definitions** specify which point types are compound and what
   fields they contain. This allows the UI and clients to decode the JSON
   payload correctly.

5. **No new mechanism needed**. The proposed JetStream encoding already supports
   data type 5 (JSON). This approach works within the existing design.

### When to use flat points vs compound points

- **Flat**: Values that change independently, need individual subscriptions, or
  are set by different sources (e.g., a sensor value vs. its calibration
  offset).
- **Compound**: Values computed together from the same source that are
  meaningless if separated (e.g., min/avg/max from an aggregation window, GPS
  lat/lon/alt, notification payloads).

### JSON vs CBOR for compound point payloads

The compound point payload needs an encoding format. The two practical options
are JSON (text) and CBOR (binary):

**JSON**

- Human-readable — easy to debug with standard tools (`jq`, browser console, log
  files).
- Universal support — every language and platform has a JSON library. No extra
  dependencies on constrained devices beyond what they likely already have.
- Larger on the wire — string keys and text encoding add overhead. A payload
  like `{"average":22.5,"peak":28.1,"rms":23.0}` is ~45 bytes.
- Already used throughout SIOT — the HTTP API, node type definitions, and
  frontend all speak JSON natively. No new encoding/decoding path needed.

**CBOR (RFC 8949)**

- Compact binary encoding — the same payload is ~30–35 bytes in CBOR,
  significant savings when multiplied across thousands of points on constrained
  links.
- Schema-less like JSON — supports the same data model (maps, arrays, strings,
  numbers, bytes) so the mental model is identical.
- Native binary and byte string support — no base64 encoding needed for binary
  sub-fields.
- Less tooling — harder to inspect without specialized tools. Debugging requires
  a CBOR decoder.
- Additional dependency — constrained devices need a CBOR library (though
  lightweight implementations exist, e.g., tinycbor, qcbor, cn-cbor).
- Go support is mature — `fxamacker/cbor` is well-maintained and mirrors the
  `encoding/json` API.

**Recommendation**: Start with JSON. It aligns with the existing SIOT stack,
minimizes new code paths, and keeps compound points easy to inspect and debug.
The size overhead is acceptable for most SIOT deployments where points flow over
NATS (typically LAN or WAN links, not severely constrained radio links). If a
future use case demands tighter payloads (e.g., LoRa, satellite, high-frequency
metering), CBOR can be added as an alternative encoding behind the existing data
type field — the point model doesn't care what bytes are in the payload, only
that sender and receiver agree on the encoding. Supporting both is
straightforward since they share the same data model.

## Consequences

- Compound points trade per-sub-field subscriptions for atomicity. Clients that
  need individual sub-fields must decode the JSON payload.
- The UI needs to understand compound point types to render them properly.
- Constrained devices sending compound points need a JSON or CBOR encoder,
  though this is minimal overhead.
- The merge model remains simple: latest timestamp wins at the point level,
  regardless of whether the point contains a float or a JSON object.

## Additional Notes/Reference

- [Golioth LightDB Stream](https://docs.golioth.io/cloud/services/lightdb-stream)
- [RFC 8428 - SenML](https://www.rfc-editor.org/rfc/rfc8428.html)
- [Sparkplug B Specification](https://sparkplug.eclipse.org/specification/version/2.2/documents/sparkplug-specification-2.2.pdf)
- [HiveMQ - Sparkplug Payload Structures](https://www.hivemq.com/blog/mqtt-payload-structures-iiot/)
- [SenML overview](https://umair-iftikhar.medium.com/senml-compact-json-for-sensor-data-with-metadata-2698efc7a915)
