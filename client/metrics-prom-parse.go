package client

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/simpleiot/simpleiot/data"
)

// This file parses the Prometheus text exposition format, version 0.0.4, which
// is what an endpoint serves when asked for text/plain. The format is a small
// line grammar, so it is read here rather than through
// github.com/prometheus/common/expfmt -- that package carries the protobuf
// exposition path and the OpenMetrics writer along with it, none of which is
// needed to read an endpoint.
//
// The parser is lenient toward the OpenMetrics variants it may meet, since an
// exporter is free to serve them: the "# EOF" terminator is ignored, and an
// exemplar following a sample is dropped.

// promMaxLine is the longest exposition line the scanner will read. Metric
// lines are far shorter than this, but an exporter is free to attach a long
// label value, and the scanner's default 64 KB limit would fail the whole
// scrape rather than the one line.
const promMaxLine = 1024 * 1024

// sample is one sample line of the exposition format, mapped onto the fields a
// point is built from. The metric name becomes the point type and needs no
// rewriting: a name matches [a-zA-Z_:][a-zA-Z0-9_:]* and none of those
// characters is rejected by data.CheckSubjectTokens. The key is rendered from
// the label set and does need it -- see renderLabels.
type sample struct {
	name    string
	key     string
	val     float64
	counter bool
}

// parseExposition reads the exposition format and returns the samples it
// carried, along with the number of lines that could not be parsed. A
// malformed line is counted and skipped rather than failing the scrape, so one
// bad line in an exporter's output does not cost us the several hundred good
// ones around it. This follows how sysTemperatures uses the readings
// host.SensorsTemperatures returns alongside its error.
func parseExposition(r io.Reader) (samples []sample, badLines int, err error) {
	// metric names declared a counter by a # TYPE line. Only these are
	// eligible for a delta point -- see promPeriodic.
	counters := make(map[string]bool)

	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), promMaxLine)

	for s.Scan() {
		line := strings.TrimLeft(s.Text(), " \t")

		if line == "" {
			continue
		}

		if line[0] == '#' {
			// "# TYPE <name> <type>" is the only comment worth reading.
			// HELP, EOF, and free-form comments are all skipped.
			if name, typ, ok := parseTypeComment(line); ok && typ == "counter" {
				counters[name] = true
			}

			continue
		}

		sm, err := parseSample(line)
		if err != nil {
			badLines++
			continue
		}

		if sm == nil {
			// a value we cannot represent as a point, such as NaN
			continue
		}

		samples = append(samples, *sm)
	}

	if err := s.Err(); err != nil {
		return samples, badLines, err
	}

	for i := range samples {
		samples[i].counter = counters[samples[i].name]
	}

	return samples, badLines, nil
}

// parseTypeComment reads a "# TYPE <name> <type>" line. Any other comment
// returns false.
func parseTypeComment(line string) (name, typ string, ok bool) {
	f := strings.Fields(line)
	if len(f) != 4 || f[1] != "TYPE" {
		return "", "", false
	}

	return f[2], f[3], true
}

// parseSample reads one sample line:
//
//	name [ "{" labels "}" ] value [ timestamp ]
//
// A nil sample with a nil error means the line parsed but carries a value that
// cannot become a point, which is a non-finite value.
func parseSample(line string) (*sample, error) {
	i := strings.IndexAny(line, "{ \t")
	if i < 0 {
		return nil, fmt.Errorf("no value in sample line %q", line)
	}

	sm := sample{name: line[:i]}
	rest := line[i:]

	if rest[0] == '{' {
		key, r, err := parseLabels(rest)
		if err != nil {
			return nil, err
		}

		sm.key, rest = key, r
	}

	// an OpenMetrics exemplar follows the value after a '#'. Everything from
	// there on describes a trace, not the sample.
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		rest = rest[:i]
	}

	// what remains is the value and an optional timestamp. The timestamp is
	// dropped: points are stamped with the scrape time, the way every other
	// client stamps what it reads.
	f := strings.Fields(rest)
	if len(f) < 1 {
		return nil, fmt.Errorf("no value in sample line %q", line)
	}

	val, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return nil, fmt.Errorf("bad value in sample line %q: %w", line, err)
	}

	if math.IsNaN(val) || math.IsInf(val, 0) {
		// Prometheus uses NaN for "no observation yet" and +Inf for an
		// unbounded value. Neither survives the trip through protobuf and
		// the store into a UI that has to render it, so the sample is
		// dropped rather than published. Note this is the sample value; a
		// le="+Inf" label is a label value and is unaffected.
		return nil, nil
	}

	sm.val = val

	return &sm, nil
}

// parseLabels reads the label set that opens s and renders it as a point key,
// returning the unconsumed remainder of the line.
//
// The set is walked a character at a time rather than split on commas, because
// a label value is quoted and may hold commas, braces, and escaped quotes --
// path="/a,b" is ordinary output that splitting would tear in half.
//
// Labels are sorted by name so the key for a series stays the same from one
// scrape to the next no matter what order the exporter emits them in, and are
// rendered as name=value pairs joined by commas. Both of those characters are
// legal in a point key. The rendered key then goes through SubjectSafeToken
// because label values are arbitrary text and routinely carry periods:
// le="0.005" appears on every histogram bucket, and the store rejects a key
// holding one.
func parseLabels(s string) (key, rest string, err error) {
	type label struct{ name, val string }

	var labels []label

	i := 1 // opening brace

	for {
		// a trailing comma before the closing brace is allowed
		for i < len(s) && (s[i] == ',' || s[i] == ' ' || s[i] == '\t') {
			i++
		}

		if i < len(s) && s[i] == '}' {
			i++
			break
		}

		eq := strings.IndexByte(s[i:], '=')
		if eq < 0 {
			return "", "", fmt.Errorf("unterminated label set in %q", s)
		}

		name := strings.TrimSpace(s[i : i+eq])
		i += eq + 1

		val, n, err := parseLabelValue(s, i)
		if err != nil {
			return "", "", err
		}

		i = n

		labels = append(labels, label{name: name, val: val})
	}

	sort.Slice(labels, func(a, b int) bool { return labels[a].name < labels[b].name })

	var b strings.Builder

	for n, l := range labels {
		if n > 0 {
			b.WriteByte(',')
		}

		b.WriteString(l.name)
		b.WriteByte('=')
		b.WriteString(l.val)
	}

	return data.SubjectSafeToken(b.String()), s[i:], nil
}

// parseLabelValue reads the quoted label value opening at s[i] and returns it
// along with the index just past its closing quote. The exposition format
// defines exactly three escapes inside a value.
func parseLabelValue(s string, i int) (string, int, error) {
	if i >= len(s) || s[i] != '"' {
		return "", 0, fmt.Errorf("unquoted label value in %q", s)
	}

	i++ // opening quote

	var val strings.Builder

	for i < len(s) {
		switch s[i] {
		case '"':
			return val.String(), i + 1, nil

		case '\\':
			i++
			if i >= len(s) {
				return "", 0, fmt.Errorf("trailing backslash in %q", s)
			}

			switch s[i] {
			case 'n':
				val.WriteByte('\n')
			case '\\', '"':
				val.WriteByte(s[i])
			default:
				// not an escape the format defines. Prometheus keeps
				// both characters rather than rejecting the line.
				val.WriteByte('\\')
				val.WriteByte(s[i])
			}

		default:
			val.WriteByte(s[i])
		}

		i++
	}

	return "", 0, fmt.Errorf("unterminated label value in %q", s)
}
