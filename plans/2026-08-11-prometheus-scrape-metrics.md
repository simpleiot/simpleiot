# Plan: Prometheus Scrape in the Metrics Client

**Branch:** `feat/prometheus-scrape` **Branched from:** `ff1e2c7d`

## Context

The metrics client (`client/metrics.go`) collects readings from the machine it
runs on: system stats through `gopsutil` and sysfs, Go runtime stats for the
SIOT process itself, and CPU and memory for a named process. Everything it
publishes is something it can read locally.

A large amount of useful data on those same machines is already exposed, but
only over HTTP in the Prometheus exposition format. Any Go service built with
`client_golang` has a `/metrics` endpoint, as do `node_exporter`, cAdvisor, and
most infrastructure written in the last decade. Today SIOT cannot see any of it.

The usual way to collect that data is to run Prometheus or `vmagent` and have it
scrape each target. For a small number of custom servers that is more friction
than it is worth: the scraper needs network reach to every target, which means
opening a port or standing up a VPN, and exposing `/metrics` to the internet
gives away a detailed picture of an application's internals to anyone who asks.

SIOT is already on the box and already holds an outbound, authenticated
connection upstream. If it scrapes `127.0.0.1`, the application binds `/metrics`
to loopback and never listens on a public interface at all. Nothing new is
exposed, no port is opened, and the readings arrive through the same sync,
store, rules, and database path as every other point. That is the case this plan
serves: a handful of custom servers, each with metrics worth collecting and no
appetite for the infrastructure a scraper normally requires.

## Design Decisions

**A new `type` on the metrics node, not a new client.** The metrics node already
carries description, period, and tags, publishes points on itself, and switches
behavior on a `type` point where `name` applies only to `type: process`. A
`prometheus` type with a `uri` that applies only to it follows the same shape.
One node per scrape target; several applications on a box means several nodes. A
separate client would only pay off with per-target auth, TLS, and relabeling,
none of which the loopback case needs, and any of it can be added later as
points on the same node.

**No new dependencies.** `prometheus/common/expfmt` is the obvious parser and it
brings `prometheus/common`, `prometheus/client_model`, and `munnerz/goautoneg`
along with it, most of which implements the protobuf exposition path and the
OpenMetrics writer. `go.mod` has no Prometheus code today. The text format is a
line-oriented grammar, and parsing it is a bounded job of roughly 150 lines
against a project that cares about dependency count and binary size on embedded
targets. Same reasoning that led the GPS client to speak gpsd directly rather
than take `go-gpsd`.

**Point type is the metric name, point key is the label set.** A Prometheus
sample is `(name, {labels}, value)` and a SIOT point is `(type, key, value)`, so
the mapping has one degree of freedom. Metric names match
`[a-zA-Z_:][a-zA-Z0-9_:]*`, and none of those characters is in
`data.invalidSubjectChars` — `:` is legal in a NATS subject token — so a metric
name is already a valid point type with no transformation at all. That makes the
metric name the thing rules and Grafana queries select on, matching how every
other client behaves.

**Label values must be sanitized; metric names must not.** Label values are
arbitrary UTF-8 and routinely carry periods: `le="0.005"` on every histogram
bucket, `quantile="0.99"` on every summary, and version strings and paths
elsewhere. Since 0.23.2 the store rejects any point whose type or key holds a
period, whitespace, or a NATS wildcard, and sets an `error` point on the node
when it does. So the rendered key goes through `data.SubjectSafeToken`, the same
treatment the client already gives kernel device names and mount points.

**Keys render as `name=value` pairs, sorted by label name.** `=` and `,` are
both subject-safe, so `method=post,code=200` survives intact and stays readable
in the UI and in a Grafana query. Sorting by label name means the key for a
given series does not change from one scrape to the next regardless of the order
the exporter emits labels in. A sample with no labels gets an empty key, which
is what the rest of the client already does for keyless points.

**The key is a label set the db client can expand.** Rendering the labels into
one key is what SIOT's point shape allows, but it is not the shape a time series
database wants: `label_replace` can pull a label back out of the key at query
time, and Grafana can group on the result, though every panel then carries a
regex and `histogram_quantile` cannot use an `le` it has to extract. Since
`name=value,name=value` is a grammar the db client can parse as reliably as the
metrics client wrote it, the db client splits the key back into individual
Influx tags, which VictoriaMetrics stores as ordinary labels. The `key` tag is
still written alongside, so a query that selects on the whole set keeps working
and a key that does not parse as a label set loses nothing. This is Phase 4, and
it is worth doing in the same branch because the scrape is much less useful
without it.

**Sanitized label values are restored only where the meaning is unambiguous.**
`SubjectSafeToken` maps a period to `_`, so the db client cannot tell
`le="0.005"` from a label value that genuinely held an underscore. Restoring
every underscore would corrupt real data, and restoring none leaves
`histogram_quantile` unusable, which is most of the reason to want buckets in a
database at all. Prometheus defines exactly two numeric label names, `le` and
`quantile`, so the db client restores periods only on those two, and only when
the value is otherwise all digits. Everything else is written as SIOT stored it.

**Scraped points may not overwrite node configuration.** This is the sharpest
edge in the design. `Run()` feeds every point arriving on the node through
`data.MergePoints` into the `Metrics` config struct, which matches on point
type. A scraped metric named `period` would rewrite the scrape interval, one
named `description` would rename the node in the UI, and one named `disabled`
would switch the client off. Prometheus naming convention makes these unlikely —
real metrics are namespaced and unit-suffixed, as in `myapp_requests_total` —
but the failure is bad enough to close rather than document. The client holds a
reserved set of point types and skips any sample whose name is in it, logging
once per name so the operator can see what was dropped and rename it.

**Counters publish a per-period delta alongside the raw value.** The metrics
worth watching on a custom server are mostly counters, and a monotonic counter
is close to useless in the SIOT UI and unusable in a SIOT rule — "alert when
errors increase" needs a rate, not a ramp. The `# TYPE` line says which samples
are counters, and the previous scrape is already in memory, so the delta is a
subtraction plus reset handling. The raw value is still published because that
is what `rate()` wants once the db client lands it in VictoriaMetrics. The delta
is published under the metric name with a `_delta` suffix, and is controlled by
a `counterDelta` point defaulting to true.

**A hard series cap, applied deterministically.** Node points live in memory,
are stored per node, and replicate over sync, so an endpoint with thousands of
series would make one node dwarf the rest of the tree. A `client_golang` default
registry plus an application's own metrics runs 50–150 series, so a cap of 200
leaves headroom for the intended case while bounding an accident. Samples are
sorted before the cap is applied, so the same series survive each scrape rather
than flapping, and the truncation is logged and reported as an `error` point
rather than passing silently.

**An optional list of metric name prefixes.** Applications namespace their
metrics, so a prefix covers the real filtering cases — `myapp_` keeps an
application's own metrics and drops the `go_*` and `promhttp_*` series the
default registry adds — without a regex engine or an allowlist to maintain. A
sample is kept when it matches any entry, which covers an application
namespacing under more than one name and a node collecting a couple of
subsystems from a larger exporter, so neither needs a second node. This is a
list point of type `prefix`, the same shape the db client's `tagPointType`
already uses. An empty list takes everything, up to the cap.

**No auth or TLS configuration in the first pass.** The target is loopback.
Adding a bearer token or a CA path later is additive, and leaving them out now
keeps the node simple.

## Mapping Examples

Everything below assumes the metrics node is scraping an application that
namespaces its metrics under `myapp_`. In each case the exposition input is on
the left and the points the node ends up holding are on the right.

### A counter with labels

```
# TYPE myapp_requests_total counter
myapp_requests_total{method="post",code="200"} 1027
```

| Point type                   | Key                    | Value       |
| ---------------------------- | ---------------------- | ----------- |
| `myapp_requests_total`       | `code=200,method=post` | 1027        |
| `myapp_requests_total_delta` | `code=200,method=post` | _see below_ |

The exporter emitted `method` before `code`; the key sorts them by label name,
so the key stays the same if a future release of the application emits them in
the other order. Nothing in either the name or the key needed sanitizing.

### A gauge with no labels

```
# TYPE myapp_queue_depth gauge
myapp_queue_depth 17
```

| Point type          | Key | Value |
| ------------------- | --- | ----- |
| `myapp_queue_depth` | ``  | 17    |

An empty key, which is what the client already publishes for keyless readings
such as `metricSysCPUPercent`. No delta point, because a gauge is not monotonic
and the reading already means what it says.

### A histogram

The exposition format has already flattened the histogram into ordinary samples
before we see it, so nothing special is needed to handle one:

```
# TYPE myapp_request_duration_seconds histogram
myapp_request_duration_seconds_bucket{le="0.005"} 24054
myapp_request_duration_seconds_bucket{le="0.05"} 33444
myapp_request_duration_seconds_bucket{le="+Inf"} 144320
myapp_request_duration_seconds_sum 53423.0
myapp_request_duration_seconds_count 144320
```

| Point type                              | Key        | Value   |
| --------------------------------------- | ---------- | ------- |
| `myapp_request_duration_seconds_bucket` | `le=0_005` | 24054   |
| `myapp_request_duration_seconds_bucket` | `le=0_05`  | 33444   |
| `myapp_request_duration_seconds_bucket` | `le=+Inf`  | 144320  |
| `myapp_request_duration_seconds_sum`    | ``         | 53423.0 |
| `myapp_request_duration_seconds_count`  | ``         | 144320  |

This is the case that makes sanitizing mandatory rather than defensive. Bucket
boundaries are decimals, and a key of `le=0.005` is rejected outright by the
store as of 0.23.2. Note that `0.005` and `0.05` remain distinct after the
substitution — `le=0_005` and `le=0_05` — which the parser tests assert
directly. `+Inf` is a label value, not a sample value, so it is carried through
untouched.

No delta points appear here. The `# TYPE` line names the base metric as a
histogram, and only a name declared `counter` is delta-eligible in this pass.
The `_bucket`, `_sum`, and `_count` series are monotonic too, so this is a
defensible thing to revisit — see Open Questions.

### A summary

```
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0.25"} 0.000103
go_gc_duration_seconds_sum 0.29
go_gc_duration_seconds_count 3016
```

| Point type                     | Key             | Value    |
| ------------------------------ | --------------- | -------- |
| `go_gc_duration_seconds`       | `quantile=0_25` | 0.000103 |
| `go_gc_duration_seconds_sum`   | ``              | 0.29     |
| `go_gc_duration_seconds_count` | ``              | 3016     |

A summary is the one shape where the base metric name itself carries a label,
which is why the mapping keys off the sample line rather than the `# TYPE`
declaration. With a `myapp_` prefix configured none of these would be collected;
they are shown because they arrive from any `client_golang` registry by default
and the parser fixture exercises them.

### Label values that need sanitizing

| Label set                                | Rendered key                       |
| ---------------------------------------- | ---------------------------------- |
| `{version="1.4.2",branch="feat/scrape"}` | `branch=feat/scrape,version=1_4_2` |
| `{path="/api/v1",agent="curl 8.5"}`      | `agent=curl_8_5,path=/api/v1`      |
| `{queue="jobs.high"}`                    | `queue=jobs_high`                  |
| `{note="a,b"}`                           | `note=a,b`                         |

Only the characters the store rejects are replaced. Slashes, colons, and hyphens
survive, which keeps paths and version constraints readable. The last row is the
reason `parseLabels` walks the label set a character at a time: a comma inside a
quoted value is data, and splitting the set on `,` would produce two malformed
labels out of one good one.

### A name that collides with node configuration

```
# TYPE period counter
period 42
```

Nothing is published, and the client logs the skip once:

```
Metrics: prometheus scrape skipping metric "period", which collides with a
node configuration point type
```

Without the reserved set this sample would merge into the `Metrics` config
struct and set the scrape interval to 42 seconds.

### Counter deltas across scrapes

Following `myapp_requests_total{method="post",code="200"}` through four scrapes,
with the application restarting between the third and fourth:

| Scrape | Raw value | `_delta` published | Why                                                                                     |
| ------ | --------- | ------------------ | --------------------------------------------------------------------------------------- |
| 1      | 1027      | none               | nothing to subtract from yet                                                            |
| 2      | 1053      | 26                 | `1053 - 1027`                                                                           |
| 3      | 1061      | 8                  | `1061 - 1053`                                                                           |
| 4      | 12        | 12                 | value went backwards, so the counter restarted and the current value is the count since |

The raw point is published on every scrape, including the first, so a database
holding the series can still compute `rate()` over it in the normal way. A
failed scrape publishes neither point and leaves the stored previous value
alone, so the delta after a recovery covers the whole gap rather than reporting
a false reset.

### Where sanitizing loses information

```
myapp_hits{path="/a b"} 1
myapp_hits{path="/a_b"} 2
```

Both render to key `path=/a_b`, so the second sample overwrites the first and
the node ends up holding a single point with value 2. This is the tradeoff the
Risks section describes, and it is the same one the client already accepts for
sysfs device names.

## Querying the Result

The db client writes every point to the `points` measurement with `type` and
`key` as tags and the node's identity and inherited tags as `node.*` tags
(`client/db.go`). VictoriaMetrics accepts that over the Influx line protocol and
names the resulting series `points_value`, so the counter from the first mapping
example arrives as:

```
points_value{type="myapp_requests_total", key="code=200,method=post",
             node.id="…", node.description="My App"}
```

Without Phase 4 the labels are still reachable, since the key is a plain label
and `label_replace` can extract from it:

```promql
sum by (method) (
  label_replace(points_value{type="myapp_requests_total_delta"},
                "method", "$1", "key", ".*method=([^,]+).*")
)
```

Sorting the labels by name is what makes that regex stable across scrapes and
across exporter releases. Selecting rather than grouping needs no regex function
at all, because a label matcher is already a regex:
`key=~".*method=post(,.*)?"`.

Two things that expression cannot do are the reason Phase 4 exists. Every panel
carries its own extraction, one `label_replace` per label it wants to group on,
which is a lot of duplicated regex in a dashboard that would otherwise be
ordinary PromQL. And `histogram_quantile` reads the `le` label as a float, so a
bucket series has to arrive with `le` already split out and already numeric; no
amount of query-time work makes `le="0_005"` acceptable to it.

With Phase 4 the same sample arrives as:

```
points_value{type="myapp_requests_total", key="code=200,method=post",
             code="200", method="post", node.id="…", node.description="My App"}
```

and the query is what a Prometheus user would have written in the first place:

```promql
sum by (method) (points_value{type="myapp_requests_total_delta"})
```

Histograms work the same way, with the period restored on `le`:

```promql
histogram_quantile(0.95,
  sum by (le) (points_value{type="myapp_request_duration_seconds_bucket"}))
```

Note that adding labels changes series identity, so a dashboard built against
scraped points before Phase 4 sees new series after it. Nothing else in SIOT is
affected, since the change is confined to what the db client writes.

## Point and Node Types

Added to `data/schema.go`, in the metrics section near `PointValueSystem`:

```go
PointValuePrometheus = "prometheus"

// PointTypeCounterDelta enables publishing a per-period delta alongside the
// raw value of each counter a Prometheus endpoint reports. A counter is
// monotonic, so the raw value answers "how many since start" while the delta
// answers "how many this period", which is the reading a rule can act on.
PointTypeCounterDelta = "counterDelta"

// PointTypeMaxSeries bounds how many samples a single scrape may publish.
PointTypeMaxSeries = "maxSeries"

// CounterDeltaSuffix is appended to a counter's metric name to form the point
// type its per-period delta is published under.
const CounterDeltaSuffix = "_delta"
```

`uri` (`data.PointTypeURI`) and `prefix` (`data.PointTypePrefix`) already exist
and are reused rather than duplicated — the db client uses the first and the
update client the second.

The point types a scrape publishes are not fixed, since they come from whatever
the endpoint exposes. That is a departure from every other client, and it is the
reason the reserved set below exists.

### Reserved point types

Defined in `client/metrics-prom.go`. A sample whose metric name matches one of
these is skipped:

- The `Metrics` config fields: `description`, `type`, `name`, `period`, `uri`,
  `prefix`, `counterDelta`, `maxSeries`
- Node-level points the manager and UI depend on: `tag`, `disabled`, `error`,
  `errorCount`, `errorCountReset`, `connected`, `debug`, `log`, `nodeType`

## Client Configuration Struct

```go
// Metrics represents the config of a metrics node type
type Metrics struct {
	ID          string `node:"id"`
	Parent      string `node:"parent"`
	Description string `point:"description"`
	Type        string `point:"type"`
	Name        string `point:"name"`
	Period      int    `point:"period"`

	// Prometheus scrape config, used when Type is prometheus
	URI          string   `point:"uri"`
	Prefixes     []string `point:"prefix"`
	CounterDelta bool     `point:"counterDelta"`
	MaxSeries    int      `point:"maxSeries"`
}
```

Example node:

```yaml
nodes:
  - metrics:
      description: My App
      type: prometheus
      uri: http://127.0.0.1:9100/metrics
      period: 30
      prefix:
        - myapp_
        - worker_
      tag:
        machine: press-1
```

## Exposition Format

The parser handles the text format, version 0.0.4, and is lenient toward the
OpenMetrics variants it may encounter.

Per line, after trimming leading whitespace:

- Blank lines are skipped.
- A line starting with `#` is a comment. `# TYPE <name> <type>` is recorded so
  counters can be identified. `# HELP` and everything else, including
  OpenMetrics' `# EOF`, is ignored.
- Anything else is a sample: `name [ "{" labels "}" ] value [ timestamp ]`.

The label set is the only part that is not trivial. Values are quoted and may
hold commas, braces, and escaped quotes, so splitting on `,` breaks on real
input — `path="/a,b"` and a URL-valued label both defeat it. The set is walked a
character at a time. The escapes inside a value are `\\`, `\"`, and `\n`, and
that is the complete list.

After the labels, the remainder is truncated at an unquoted `#` so OpenMetrics
exemplars fall away, then split with `strings.Fields` into value and optional
timestamp. The timestamp is dropped; scrape time is what gets published, which
matches how every other client stamps its points.

`strconv.ParseFloat` accepts `NaN`, `+Inf`, and `-Inf` directly. A non-finite
value is skipped rather than published — it would travel through protobuf into
the store and reach the UI as something nothing downstream renders. Note that
`le="+Inf"` is a label _value_ and is unaffected; it belongs in the key.

`bufio.Scanner` caps a token at 64 KB by default. Metric lines are far shorter
than that, but a `Buffer` call raising it to 1 MB is one line and removes a
failure mode that would only appear on someone else's exporter.

```go
// sample is one line of the exposition format
type sample struct {
	name    string  // valid as a point type with no transformation
	key     string  // rendered from the sorted label set, subject-safe
	val     float64
	counter bool    // from the # TYPE line
}

func parseExposition(r io.Reader) ([]sample, error)
```

Parsing stays a pure function over an `io.Reader`, so its tests need no HTTP.

## Implementation Plan

### Phase 1: Exposition Parser

**Goal:** Turn the text format into samples, with no NATS and no HTTP.

1. Create `client/metrics-prom-parse.go` with `parseExposition`, `parseLabels`,
   and the `sample` type described above.
2. `parseLabels` returns the rendered key and the unconsumed remainder of the
   line. It sorts by label name, renders `name=value` joined by `,`, and passes
   the result through `data.SubjectSafeToken`.
3. A malformed line is counted and skipped rather than failing the whole scrape.
   One bad line in an exporter's output should not cost the other several
   hundred. `parseExposition` returns the samples it read along with the count
   of lines it could not parse, following the pattern `sysTemperatures` uses
   with `host.SensorsTemperatures`.

**Verify:** `go build ./... && go test -race ./client/`

**Test:** Fixtures under `client/testdata/`, captured rather than hand-written:
`prom-client-golang.txt` from a Go service, which supplies histograms, summaries
with `quantile` labels, and the `go_*` and `promhttp_*` series for free, and
`prom-node-exporter.txt` for awkward label values. Every worked example in
[Mapping Examples](#mapping-examples) is a table row in these tests, since the
examples were chosen to cover the cases the mapping can get wrong. Table-driven
cases for the grammar:

- A bare sample with no labels yields an empty key.
- A single label and a multi-label sample render as expected, with labels sorted
  by name regardless of input order.
- A label value holding a comma, and one holding a brace, are not split on.
- An escaped quote inside a value round-trips.
- A histogram bucket's `le="0.005"` renders as `le=0_005`, and `le="0.05"` as
  `le=0_05`, so the two stay distinct after sanitizing. Every point a fixture
  produces passes `CheckSubjectTokens`; this is the assertion that guards
  against the 0.23.2 store rejection.
- A `le="+Inf"` bucket is published, with `+Inf` in the key.
- A sample whose _value_ is `NaN`, `+Inf`, or `-Inf` is skipped.
- An explicit timestamp is parsed and discarded.
- An OpenMetrics exemplar suffix is stripped and does not corrupt the value.
- `# EOF`, `# HELP` with escaped characters, and a free-form comment are all
  ignored.
- `# TYPE x counter` marks `x` as a counter and leaves `x_sum` unmarked.
- A malformed line is counted and does not prevent the surrounding lines from
  parsing.

### Phase 2: Scrape and Publish

**Goal:** Fetch an endpoint on a period and publish its samples as points.

1. Create `client/metrics-prom.go` with `func (m *MetricsClient) promPeriodic()`
   and add `case data.PointValuePrometheus` to the `Run()` type switch.
2. Add the four config fields to `Metrics`. Default `MaxSeries` to 200 and
   `CounterDelta` to true when unset, publishing the defaults back to the node
   the way `checkPeriod` already does, so the values are visible and editable in
   the UI rather than implicit.
3. Fetch with a package-level `http.Client` whose timeout is
   `min(10s, period/2)`, so a slow endpoint cannot leave scrapes overlapping.
   Set `Accept: text/plain;version=0.0.4;q=1,*/*;q=0.1` — `client_golang` serves
   protobuf or OpenMetrics when asked, and asking for the text format keeps the
   input shape stable. Leave `Accept-Encoding` alone; `http.Transport` adds gzip
   and decompresses transparently. Wrap the body in an `io.LimitReader` at 8 MB.
4. Filter: drop samples whose name carries none of the configured prefixes,
   ignoring empty entries so a prefix added in the UI but not yet filled in does
   not quietly widen the filter. Then drop samples whose name is in the reserved
   set, logging each reserved name once.
5. Sort the remaining samples by name then key and truncate at `MaxSeries`.
6. Counter deltas: hold `map[string]float64` on the client keyed by
   `name + "\x00" + key`. For a counter, publish the raw value, and when a
   previous value exists publish `name + CounterDeltaSuffix`:

   ```go
   // a counter that went backwards means the process restarted, so the
   // current value is the delta since that restart
   delta := cur - prev
   if cur < prev {
       delta = cur
   }
   ```

   No delta is published on the first scrape of a series, since there is nothing
   to subtract from. Gauges pass through untouched.

7. On a failed scrape — connection refused, timeout, non-200 — log, set an
   `error` point on the node, and publish nothing. Stale values are worse than
   absent ones. Clear the `error` point on the next success. The counter state
   map is left alone across a failure, so a restarted scrape does not produce a
   spurious delta.
8. Report a truncated scrape the same way, through the `error` point, so it is
   visible in the UI and not only in the log.

**Verify:** `go build ./... && go test -race ./client/`, then against a live
endpoint — SIOT's own NATS server exposes one, and `siot_run` with a metrics
node pointed at it is an end-to-end check that costs nothing to set up.

**Test:** An `httptest.Server` covering the fetch path, and pure functions for
the rest:

- A served fixture publishes the expected point types and keys.
- Every published point passes `CheckSubjectTokens`.
- A `prefix` keeps the matching metrics and drops `go_*` and `promhttp_*`, and a
  second prefix keeps what it matches as well.
- A metric named `period` is skipped and does not alter the configured period.
  Same for `description` and `disabled`. This is the reserved-set test.
- Two scrapes of a counter produce a delta equal to the difference, and none on
  the first scrape.
- A counter that decreases between scrapes yields a delta equal to the new
  value, not a negative number.
- A gauge produces no delta point.
- `counterDelta` false suppresses delta points entirely.
- An endpoint over `maxSeries` publishes exactly `maxSeries` points, the same
  ones on a repeat scrape, and sets an `error` point.
- A non-200 response, a connection refused, and a timeout each set an `error`
  point and publish no metrics; a subsequent success clears it.
- A response larger than the limit is truncated rather than read into memory.

### Phase 3: Frontend

**Goal:** Configure a scrape from the UI.

1. `frontend/src/Components/NodeMetrics.elm`: add
   `( Point.valuePrometheus, "prometheus" )` to the type option list, and gate
   `uri`, `counterDelta`, and `maxSeries` inputs, and a `prefix` list input,
   behind `viewIf (metricsType == Point.valuePrometheus)`, the way `name` is
   already gated behind `valueProcess`.
2. `frontend/src/Api/Point.elm`: add `valuePrometheus`, `typeCounterDelta`, and
   `typeMaxSeries`. `typeURI` and `typePrefix` already exist.
3. No additions to `metricFormaters` — scraped point types are not known ahead
   of time, so they fall through to `Point.renderPoint2`, which renders the
   type, key, and value. That is the right display for a metric whose units the
   UI cannot know.

**Verify:** `siot_build_frontend`,
`cd frontend && npx elm-review && npx elm-test`

### Phase 4: Label Expansion in the db Client

**Goal:** Write each label in a scraped key as its own database label, so a
scraped series queries like the Prometheus series it came from.

1. Create `client/db-key-labels.go` with
   `func expandKeyLabels(key string) map[string]string`, a pure function over
   the key with no NATS and no database. It returns nil for a key that is not a
   label set, which the caller treats as "add nothing".
2. Parsing rules, chosen so that a key from any other client is left alone:
   - Split on `,`. Every chunk must be `name=value` with `name` matching
     `[a-zA-Z_][a-zA-Z0-9_]*`.
   - If any chunk fails, return nil. The expansion is all or nothing, so a key
     is never half interpreted. This is also what handles the `note="a,b"` case
     from [Mapping Examples](#label-values-that-need-sanitizing): a comma inside
     a label value makes the key unsplittable, the strict rule declines it, and
     the `key` tag still carries the whole set.
   - An empty value is dropped rather than written, matching Prometheus, where
     an empty label and an absent label are the same thing.
   - Restore periods on `le` and `quantile` when the value is otherwise only
     digits and underscores, so `le=0_005` is written as `le="0.005"` and
     `histogram_quantile` can read it. `le=+Inf` needs no restoration and gets
     none. No other label name is rewritten.
3. Skip a label named `type` or `key`, which are the tags the db client writes
   itself, logging each skipped name once per client. A `node.*` tag cannot
   collide, since a Prometheus label name has no period in it.
4. Factor the tag map both write paths build (`db.go`, in the HR subscription
   and in the point handler) into one
   `func (dbc *DbClient) pointTags(nodeID string, pt data.Point) map[string]string`
   so the two cannot drift, and call `expandKeyLabels` from it.
5. Add `ExpandKeyLabels bool` with `point:"expandKeyLabels"` to the `Db` config
   struct and `PointTypeExpandKeyLabels` to `data/schema.go`. Default it to
   true, published back to the node the way Phase 2 publishes its defaults, so
   the setting is visible and can be turned off. The strict parse means the
   default costs nothing for the keys every other client writes.
6. Frontend: `frontend/src/Components/NodeDb.elm` gets a checkbox, and
   `frontend/src/Api/Point.elm` gets `typeExpandKeyLabels`.

**Verify:** `go build ./... && go test -race ./client/`, then a scrape into a
local VictoriaMetrics with a `histogram_quantile` query over the bucket series.

**Test:** Table-driven over `expandKeyLabels`, plus the write path:

- `code=200,method=post` yields both labels, and the `key` tag is still written.
- An empty key, `eth0`, and `/dev/sda` yield no labels, which covers the keys
  the rest of SIOT publishes.
- `note=a,b` yields no labels rather than a partial expansion.
- `le=0_005` yields `le="0.005"`, `quantile=0_99` yields `quantile="0.99"`, and
  `le=+Inf` is unchanged.
- `version=1_4_2` is unchanged, since only `le` and `quantile` are restored.
- `type=foo,method=post` writes `method` and leaves the `type` tag as the point
  type.
- `expandKeyLabels` false writes only the tags the client writes today.
- The HR path and the point path produce the same tags for the same point.

### Phase 5: Documentation and Changelog

1. `docs/user/metrics.md`: a "Prometheus Metrics" section covering the loopback
   argument, the name-to-point-type mapping, label sanitizing, counter deltas
   and why the raw counter is kept alongside, the series cap, and the reserved
   names. Include the YAML schema example, matching the existing schema section
   at the end of that page. Include the example queries from
   [Querying the Result](#querying-the-result), since the mapping is only worth
   as much as the query a reader can write from it.
2. `docs/user/database.md`: document `expandKeyLabels`, what makes a key a label
   set, the `le` and `quantile` restoration, and the note that turning it on
   changes series identity for points whose keys expand.
3. `CHANGELOG.md` under `## Next`: an `### Added` entry.
4. `CLAUDE.md`: no change needed — the client list there is illustrative and
   already names Metrics.
5. Update `plans/plans.md` with the final status.

**Verify:** `siot_test`

## Files Touched

| File                                      | Change                                                                                                                  |
| ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `data/schema.go`                          | `PointValuePrometheus`, `PointTypeCounterDelta`, `PointTypeMaxSeries`, `CounterDeltaSuffix`, `PointTypeExpandKeyLabels` |
| `client/metrics.go`                       | Four config fields, `prometheus` case in the `Run()` switch, defaults                                                   |
| `client/metrics-prom.go`                  | new — scrape, filter, cap, counter deltas, error points                                                                 |
| `client/metrics-prom-parse.go`            | new — exposition format parser                                                                                          |
| `client/metrics-prom_test.go`             | new — `httptest` and mapping tests                                                                                      |
| `client/metrics-prom-parse_test.go`       | new — grammar tests                                                                                                     |
| `client/testdata/prom-client-golang.txt`  | new — fixture                                                                                                           |
| `client/testdata/prom-node-exporter.txt`  | new — fixture                                                                                                           |
| `client/db.go`                            | `ExpandKeyLabels` config field, shared `pointTags` for both write paths, default                                        |
| `client/db-key-labels.go`                 | new — key to label set expansion                                                                                        |
| `client/db-key-labels_test.go`            | new — expansion tests                                                                                                   |
| `frontend/src/Components/NodeMetrics.elm` | prometheus type option and its inputs                                                                                   |
| `frontend/src/Components/NodeDb.elm`      | `expandKeyLabels` checkbox                                                                                              |
| `frontend/src/Api/Point.elm`              | four new constants                                                                                                      |
| `docs/user/metrics.md`                    | Prometheus section                                                                                                      |
| `docs/user/database.md`                   | key label expansion                                                                                                     |
| `CHANGELOG.md`                            | Added entry                                                                                                             |

Roughly 250 lines of implementation and a similar amount of test. No change to
`go.mod`.

## Risks

**Sanitizing can merge two distinct series.** Two label sets differing only in a
character that maps to `_` — `path="/a b"` and `path="/a_b"` — collapse to one
key, and the later sample overwrites the earlier. This is the same tradeoff the
client already accepts for sysfs and interface names. It is worth a note in the
docs rather than machinery to detect it.

**Points accumulate as label sets change.** A point is stored on the node until
tombstoned, so an application that changes its label values — a `version` label
across deploys is the obvious case — grows the node over time with series that
no longer update. The cap bounds a single scrape, not the history. Worth
documenting that a `version`-style label on a scraped metric is a poor idea, and
worth watching before adding anything more clever.

**Label expansion changes series identity.** A series that carried only `type`
and `key` gains a label per label in the key, which a time series database reads
as a different series. Anyone who built a dashboard against scraped points
before Phase 4 sees the old series stop and a new one start. The window where
that matters is small, since both land in the same commit range, and
`expandKeyLabels` can be turned off on a node that has history worth keeping
continuous.

**Point types are no longer a closed set.** Every other client publishes types
declared in `data/schema.go`; a scrape publishes whatever the endpoint names.
The reserved set closes the dangerous case, but anything in the future that
assumes point types are enumerable should know this node type exists.

## Open Questions

**Should the reserved set be a rename rather than a skip?** Prefixing a
colliding name — `period` becoming `prom_period` — keeps the data instead of
dropping it. Skipping is proposed because a collision means the metric is badly
named, and a log line the operator acts on beats a silent rewrite. Easy to
revisit if it proves annoying in practice.

**Is `_delta` the right suffix?** It reads well in a Grafana query
(`points{type="http_requests_total_delta"}`) and cannot be confused with a
Prometheus convention, since `_total`, `_sum`, `_count`, and `_bucket` are the
reserved ones. The alternative — keeping the metric name as the point type and
marking the delta in the key — makes the key mean two different things depending
on the point, which seems worse.

**Should histogram and summary components get deltas?** `_bucket`, `_sum`, and
`_count` are all monotonic, so a per-period delta is as meaningful for them as
for a plain counter — "requests under 5 ms this period" is a useful reading. The
plan marks only names declared `counter` because a histogram with twelve buckets
is fourteen series, and deltas would make it twenty-eight against a cap of 200.
Deriving counter-ness from a `histogram` or `summary` type line is a few lines
of code if the cap turns out to be generous enough in practice.

**Should the key encode periods reversibly instead of restoring them?** `~` is
subject-safe, so rendering `le="0.005"` as `le=0~005` would survive the store
and decode back exactly, with no per-label-name special case in the db client
and no ambiguity for version strings or paths. It costs readability in the UI,
where `le=0~005` reads as neither the original nor a plain underscore, and it
needs an escape for a label value that genuinely holds `~`. The plan restores
`le` and `quantile` instead because those are the only two labels a query
interprets numerically, but this is worth revisiting if a third such label turns
up. (NO, lets keep it simple)

**Should expansion be gated on the source node type rather than parsed?** The db
client has the node type in its cache already, so it could expand keys only for
points from a metrics node and skip the grammar check entirely. Parsing is
proposed because it keeps `expandKeyLabels` meaningful for any client that
adopts the same key convention later, and the strict all-or-nothing rule already
declines every key the rest of SIOT writes.

**Does anyone want the inverse?** Exposing a SIOT `/metrics` endpoint so an
existing Prometheus can pull node points is a different, smaller feature, and
some people asking for Prometheus support want that one. It is not in this plan
and does not conflict with it. (not is this plan)
