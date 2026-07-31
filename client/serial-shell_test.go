package client

import (
	"strings"
	"testing"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// swapVerb turns a `p` write command into the `pt ` report the MCU would emit,
// so a formatted point can be fed straight back into the parser.
func swapVerb(line string) string {
	return shellVerbReport + strings.TrimPrefix(line, shellVerbWrite)
}

func TestShellRoundTripDataTypes(t *testing.T) {
	stamp := time.Date(2026, 7, 31, 12, 0, 0, 123456789, time.UTC)

	tests := []struct {
		name string
		p    data.Point
	}{
		{"float", data.NewPointFloat("chargerVbat", "0", 16.25)},
		{"float negative", data.NewPointFloat("chargerIbat", "0", -1.5)},
		{"int", data.NewPointInt("uptime", "0", 3600)},
		{"int large", data.NewPointInt("bootCount", "0", 2147483647)},
		{"int negative", data.NewPointInt("temp", "0", -40)},
		{"string", data.NewPointString("board", "0", "nucleo_h743zi")},
		{"string with spaces", data.NewPointString("description", "0", "lab bench H7")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.p
			p.Time = stamp

			line, err := formatPointWrite(p)
			if err != nil {
				t.Fatalf("format error: %v", err)
			}

			got, kind, err := parseShellLine(swapVerb(line))
			if err != nil {
				t.Fatalf("parse error for %q: %v", line, err)
			}
			if kind != linePoint {
				t.Fatalf("got kind %v, expected linePoint for %q", kind, line)
			}

			if got.Type != p.Type {
				t.Errorf("type: got %q, expected %q", got.Type, p.Type)
			}
			if got.DataType != p.DataType {
				t.Errorf("dataType: got %v, expected %v", got.DataType, p.DataType)
			}
			if !got.Time.Equal(stamp) {
				t.Errorf("time: got %v, expected %v", got.Time, stamp)
			}
			// compare through the accessors, since PutInt uses a
			// variable width encoding
			if got.Val() != p.Val() || got.Txt() != p.Txt() {
				t.Errorf("value: got val=%v txt=%q, expected val=%v txt=%q",
					got.Val(), got.Txt(), p.Val(), p.Txt())
			}
		})
	}
}

func TestShellJSONDataType(t *testing.T) {
	p := data.Point{Type: "cfg", Key: "0", DataType: data.PointDataTypeJSON}
	p.Data = []byte(`{"a":1}`)

	line, err := formatPointWrite(p)
	if err != nil {
		t.Fatalf("format error: %v", err)
	}

	got, kind, err := parseShellLine(swapVerb(line))
	if err != nil {
		t.Fatalf("parse error for %q: %v", line, err)
	}
	if kind != linePoint {
		t.Fatalf("got kind %v, expected linePoint", kind)
	}
	if got.DataType != data.PointDataTypeJSON {
		t.Errorf("got dataType %v, expected JSON", got.DataType)
	}
	if string(got.Data) != `{"a":1}` {
		t.Errorf("got data %q, expected %q", string(got.Data), `{"a":1}`)
	}
}

func TestShellTimeFormatIsFixedWidth(t *testing.T) {
	// This is the time.RFC3339Nano trimming trap. RFC3339Nano would render
	// these as "...00Z", "...00.1Z" and "...00.123456789Z", which is not
	// canonical and does not sort. Nine digits always.
	tests := []struct {
		name string
		ns   int
		exp  string
	}{
		{"whole second", 0, "2026-07-31T12:00:00.000000000Z"},
		{"tenth", 100000000, "2026-07-31T12:00:00.100000000Z"},
		{"full nanoseconds", 123456789, "2026-07-31T12:00:00.123456789Z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatShellTime(time.Date(2026, 7, 31, 12, 0, 0, tc.ns, time.UTC))
			if got != tc.exp {
				t.Errorf("got %q, expected %q", got, tc.exp)
			}
		})
	}
}

func TestShellTimeAlwaysUTC(t *testing.T) {
	zone := time.FixedZone("IST", 5*3600+1800)
	local := time.Date(2026, 7, 31, 17, 30, 0, 0, zone)

	got := formatShellTime(local)
	if got != "2026-07-31T12:00:00.000000000Z" {
		t.Errorf("got %q, expected the UTC rendering", got)
	}
}

func TestShellParseAcceptsShorterTimeForms(t *testing.T) {
	// The formatter is strict, but a hand-typed command or a different
	// implementation may produce a trimmed fraction.
	exp := time.Date(2026, 7, 31, 12, 0, 0, 500000000, time.UTC)

	for _, s := range []string{
		"2026-07-31T12:00:00.500000000Z",
		"2026-07-31T12:00:00.5Z",
	} {
		p, _, err := parseShellLine("pt uptime 0 INT 5 " + s)
		if err != nil {
			t.Fatalf("parse error for %q: %v", s, err)
		}
		if !p.Time.Equal(exp) {
			t.Errorf("%q: got %v, expected %v", s, p.Time, exp)
		}
	}

	// no fraction at all
	p, _, err := parseShellLine("pt uptime 0 INT 5 2026-07-31T12:00:00Z")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !p.Time.Equal(time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("got %v for a no-fraction time", p.Time)
	}
}

func TestShellNoTimeFieldLeavesTimeZero(t *testing.T) {
	p, kind, err := parseShellLine("pt uptime 0 INT 3600")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if kind != linePoint {
		t.Fatalf("got kind %v, expected linePoint", kind)
	}
	if !p.Time.IsZero() {
		t.Errorf("got time %v, expected zero so the caller stamps it", p.Time)
	}
}

func TestShellKeyConvention(t *testing.T) {
	// the MCU uses "0" for a keyless point, SIOT uses ""
	p, _, err := parseShellLine("pt uptime 0 INT 5")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if p.Key != "" {
		t.Errorf("got key %q, expected \"\" for the MCU's \"0\"", p.Key)
	}

	line, err := formatPointWrite(data.NewPointInt("uptime", "", 5))
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if line != "p uptime 0 INT 5" {
		t.Errorf("got %q, expected a blank key to encode as 0", line)
	}

	// a real key survives in both directions
	p, _, err = parseShellLine("pt fanSpeed 2 INT 1200")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if p.Key != "2" {
		t.Errorf("got key %q, expected \"2\"", p.Key)
	}
}

func TestShellQuoting(t *testing.T) {
	tests := []struct {
		name  string
		value string
		quote bool
	}{
		{"plain", "nucleo_h743zi", false},
		{"space", "lab bench H7", true},
		{"double quote", `say "hi"`, true},
		{"backslash", `a\b`, true},
		{"empty", "", true},
		{"newline", "a\nb", true},
		{"tab", "a\tb", true},
		{"punctuation needs no quotes", "192.168.1.50", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := quoteField(tc.value)
			if strings.HasPrefix(q, `"`) != tc.quote {
				t.Errorf("quoteField(%q) = %q, expected quoted=%v", tc.value, q, tc.quote)
			}

			fields, err := splitFields("p x 0 STR " + q)
			if err != nil {
				t.Fatalf("splitFields error for %q: %v", q, err)
			}
			if len(fields) != 5 {
				t.Fatalf("got %d fields %q, expected 5", len(fields), fields)
			}
			if fields[4] != tc.value {
				t.Errorf("round trip: got %q, expected %q", fields[4], tc.value)
			}
		})
	}
}

func TestShellQuotingDoesNotAddNoise(t *testing.T) {
	// a value that needs no quoting must not acquire any
	if got := quoteField("nucleo_h743zi"); got != "nucleo_h743zi" {
		t.Errorf("got %q, expected the value unchanged", got)
	}
}

func TestShellUnterminatedQuote(t *testing.T) {
	_, _, err := parseShellLine(`pt description 0 STR "unterminated`)
	if err == nil {
		t.Error("expected an error for an unterminated quote, got nil")
	}
}

func TestShellClassifyConsoleNoise(t *testing.T) {
	tests := []struct {
		name string
		line string
		kind lineKind
	}{
		{"zephyr log inf", "[00:00:00.310,000] <inf> siot: Network connected", lineLog},
		{"zephyr log err", "[00:00:12.345,000] <err> net_if: Failed", lineLog},
		{"zephyr log dbg", "[00:01:02.003,000] <dbg> z_point: merged", lineLog},
		{"boot banner", "*** Booting Zephyr OS build v4.0.0 ***", lineOther},
		{"uptime shell output", "Interface eth0 (0x24000f00)", lineOther},
		{"bare prompt text", "siot stream on", lineOther},
		{"command echo of p", "p uptime 0 INT 5", lineOther},
		{"point line", "pt uptime 0 INT 5", linePoint},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, kind, err := parseShellLine(tc.line)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.line, err)
			}
			if kind != tc.kind {
				t.Errorf("got kind %v, expected %v for %q", kind, tc.kind, tc.line)
			}
		})
	}
}

func TestShellMalformedPointLines(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"too few fields", "pt uptime 0 INT"},
		{"unknown data type", "pt uptime 0 XXX 5"},
		{"bad int", "pt uptime 0 INT notanumber"},
		{"bad float", "pt temp 0 FLT notanumber"},
		{"empty type", `pt "" 0 INT 5`},
		{"bad time", "pt uptime 0 INT 5 not-a-time"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, kind, err := parseShellLine(tc.line)
			if err == nil {
				t.Errorf("expected an error for %q", tc.line)
			}
			if kind != linePoint {
				t.Errorf("got kind %v, expected linePoint so it counts as an error", kind)
			}
		})
	}
}

func TestShellFloatNoExponentNotation(t *testing.T) {
	// the firmware's atof does not handle exponents, and they are
	// unreadable at a console
	p := data.NewPointFloat("tiny", "0", 0.0000001)
	line, err := formatPointWrite(p)
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	if strings.ContainsAny(line, "eE") {
		t.Errorf("got %q, expected no exponent notation", line)
	}
}

func TestShellCommandLengthLimit(t *testing.T) {
	p := data.NewPointString("description", "0", strings.Repeat("x", 300))
	_, err := formatPointWrite(p)
	if err == nil {
		t.Error("expected an error for a command over the MCU shell buffer limit")
	}
}

func TestShellTruncationWarning(t *testing.T) {
	tests := []struct {
		name string
		p    data.Point
		warn bool
	}{
		{"short enough", data.NewPointString("board", "0", "nucleo"), false},
		{"long data", data.NewPointString("board", "0", strings.Repeat("x", 25)), true},
		{"long type", data.NewPointInt(strings.Repeat("t", 30), "0", 1), true},
		{"long key", data.NewPointInt("x", strings.Repeat("k", 25), 1), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mcuWouldTruncate(tc.p) != ""
			if got != tc.warn {
				t.Errorf("got warn=%v, expected %v", got, tc.warn)
			}
		})
	}
}
