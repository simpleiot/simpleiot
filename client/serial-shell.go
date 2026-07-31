package client

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// The Zephyr shell protocol exchanges points as lines of ASCII. Both
// directions use the same field layout and differ only in the verb:
//
//	pt <type> <key> <INT|FLT|STR|JSN> <data> [<time>]   MCU to MPU
//	p  <type> <key> <INT|FLT|STR|JSN> <data> [<time>]   MPU to MCU
//
// `p` is the command the Zephyr firmware already registers, so anything SIOT
// writes could have been typed by hand. `pt` is deliberately different so an
// echoed command is never mistaken for a point report.
const (
	shellVerbWrite  = "p"
	shellVerbReport = "pt"
)

// shellTimeLayout is RFC 3339 UTC with a fixed nine digit fractional part.
// time.RFC3339Nano is deliberately not used: it trims trailing zeros, which
// makes the encoding non-canonical and means the strings do not sort. A fixed
// width is reproducible by any correct formatter on either end.
const shellTimeLayout = "2006-01-02T15:04:05.000000000Z07:00"

// shellMaxCmdLen matches Zephyr's CONFIG_SHELL_CMD_BUFF_SIZE default. A longer
// command would be silently truncated by the shell, so we refuse to send it.
const shellMaxCmdLen = 256

// MCU point field widths, from the firmware's point struct in include/point.h.
// SIOT truncates nothing on receive, but warns when sending a point the MCU
// would quietly shorten.
const (
	mcuMaxTypeLen = 24
	mcuMaxKeyLen  = 20
	mcuMaxDataLen = 20
)

// lineKind classifies a line received from the MCU console.
type lineKind int

const (
	// lineOther is anything not recognized: banners, command output, blank
	// lines. Not an error; a console is expected to carry other traffic.
	lineOther lineKind = iota
	// linePoint is a well formed `pt ` line.
	linePoint
	// lineLog is a Zephyr log message, forwarded as a log point.
	lineLog
)

// zephyrLogRe matches a Zephyr log line, e.g.
// "[00:00:12.345,000] <inf> siot: Network connected"
var zephyrLogRe = regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\.\d{3},\d{3}\] <(err|wrn|inf|dbg)> `)

// quoteField renders a single field for the Zephyr shell tokenizer. A value is
// emitted bare unless it contains a space, a quote, a backslash, or a control
// character, in which case it is double quoted with backslash escaping. An
// empty value is quoted so it remains a distinct argument.
func quoteField(s string) string {
	if s == "" {
		return `""`
	}

	needsQuote := false
	for _, r := range s {
		if r == ' ' || r == '"' || r == '\\' || r == '\t' || r == '\r' || r == '\n' || r < 0x20 {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}

	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// splitFields tokenizes a line using the same quoting rules quoteField
// produces. Used for both directions so there is one set of rules.
func splitFields(line string) ([]string, error) {
	var (
		fields  []string
		cur     strings.Builder
		inField bool
		inQuote bool
		escaped bool
	)

	for _, r := range line {
		switch {
		case escaped:
			switch r {
			case 'r':
				cur.WriteByte('\r')
			case 'n':
				cur.WriteByte('\n')
			case 't':
				cur.WriteByte('\t')
			default:
				cur.WriteRune(r)
			}
			escaped = false
		case r == '\\' && inQuote:
			escaped = true
		case r == '"':
			inQuote = !inQuote
			inField = true
		case (r == ' ' || r == '\t') && !inQuote:
			if inField {
				fields = append(fields, cur.String())
				cur.Reset()
				inField = false
			}
		default:
			cur.WriteRune(r)
			inField = true
		}
	}

	if inQuote {
		return nil, errors.New("unterminated quote")
	}
	if escaped {
		return nil, errors.New("trailing escape")
	}
	if inField {
		fields = append(fields, cur.String())
	}

	return fields, nil
}

// dataTypeToShell maps a point data type to its three letter wire code.
func dataTypeToShell(dt data.PointDataType) (string, error) {
	switch dt {
	case data.PointDataTypeFloat:
		return "FLT", nil
	case data.PointDataTypeInt:
		return "INT", nil
	case data.PointDataTypeString:
		return "STR", nil
	case data.PointDataTypeJSON:
		return "JSN", nil
	}
	return "", fmt.Errorf("unsupported point data type: %v", dt)
}

// formatShellTime renders a timestamp in the canonical wire form. Always UTC,
// so the MCU parser never has to handle a numeric zone offset.
func formatShellTime(t time.Time) string {
	return t.UTC().Format(shellTimeLayout)
}

// parseShellTime accepts the canonical form and also the shorter forms a
// hand-typed command or a different formatter may produce. Only the formatter
// is strict.
func parseShellTime(s string) (time.Time, error) {
	for _, layout := range []string{shellTimeLayout, time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time: %v", s)
}

// pointDataToShell renders the point value as the string that goes on the
// wire, before quoting.
func pointDataToShell(p data.Point) (string, error) {
	switch p.DataType {
	case data.PointDataTypeFloat:
		v, err := p.ValueFloat()
		if err != nil {
			return "", err
		}
		// 'f' rather than 'g': the firmware's atof does not handle exponent
		// notation, and it is unreadable at a console besides
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case data.PointDataTypeInt:
		v, err := p.ValueInt()
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(v, 10), nil
	case data.PointDataTypeString, data.PointDataTypeJSON:
		return string(p.Data), nil
	}
	return "", fmt.Errorf("unsupported point data type: %v", p.DataType)
}

// formatPointWrite renders a point as the `p` command that sets it on the MCU.
// The timestamp is always included when the point has one: carrying SIOT's
// stamp through the MCU and back is what makes the echo identifiable.
func formatPointWrite(p data.Point) (string, error) {
	dt, err := dataTypeToShell(p.DataType)
	if err != nil {
		return "", err
	}

	d, err := pointDataToShell(p)
	if err != nil {
		return "", err
	}

	key := p.Key
	if key == "" {
		// the firmware normalizes a blank key to "0"; do it here so neither
		// side has to know the other's convention
		key = "0"
	}

	fields := []string{
		shellVerbWrite,
		quoteField(p.Type),
		quoteField(key),
		dt,
		quoteField(d),
	}
	if !p.Time.IsZero() {
		fields = append(fields, formatShellTime(p.Time))
	}

	line := strings.Join(fields, " ")
	if len(line) > shellMaxCmdLen {
		return "", fmt.Errorf("command is %v bytes, over the MCU shell limit of %v: %v",
			len(line), shellMaxCmdLen, p.Type)
	}

	return line, nil
}

// mcuWouldTruncate reports whether the MCU's fixed width point fields would
// silently shorten this point, which is hard to diagnose from the far end.
func mcuWouldTruncate(p data.Point) string {
	if len(p.Type) >= mcuMaxTypeLen {
		return fmt.Sprintf("type %q exceeds the MCU's %v byte field", p.Type, mcuMaxTypeLen)
	}
	if len(p.Key) >= mcuMaxKeyLen {
		return fmt.Sprintf("key %q exceeds the MCU's %v byte field", p.Key, mcuMaxKeyLen)
	}
	if p.DataType == data.PointDataTypeString || p.DataType == data.PointDataTypeJSON {
		if len(p.Data) >= mcuMaxDataLen {
			return fmt.Sprintf("data %q exceeds the MCU's %v byte field",
				string(p.Data), mcuMaxDataLen)
		}
	}
	return ""
}

// parseShellLine classifies a line received from the MCU console and, for a
// point line, decodes it. Lines that are neither points nor logs are not
// errors: a Zephyr console legitimately carries banners and command output.
func parseShellLine(line string) (data.Point, lineKind, error) {
	if zephyrLogRe.MatchString(line) {
		return data.Point{}, lineLog, nil
	}

	if line != shellVerbReport && !strings.HasPrefix(line, shellVerbReport+" ") {
		return data.Point{}, lineOther, nil
	}

	fields, err := splitFields(line)
	if err != nil {
		return data.Point{}, linePoint, fmt.Errorf("point line: %w", err)
	}

	// verb, type, key, data type, data, and an optional time
	if len(fields) < 5 {
		return data.Point{}, linePoint,
			fmt.Errorf("point line has %v fields, expected at least 5", len(fields))
	}

	p := data.Point{Type: fields[1], Key: fields[2]}
	if p.Type == "" {
		return data.Point{}, linePoint, errors.New("point line has an empty type")
	}
	if p.Key == "0" {
		// MCU convention for a keyless point
		p.Key = ""
	}

	switch fields[3] {
	case "FLT":
		v, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			return data.Point{}, linePoint, fmt.Errorf("invalid FLT data %q: %w", fields[4], err)
		}
		p.PutFloat(v)
	case "INT":
		v, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil {
			return data.Point{}, linePoint, fmt.Errorf("invalid INT data %q: %w", fields[4], err)
		}
		p.PutInt(v)
	case "STR":
		p.PutString(fields[4])
	case "JSN":
		p.DataType = data.PointDataTypeJSON
		p.Data = []byte(fields[4])
	default:
		return data.Point{}, linePoint, fmt.Errorf("unknown data type %q", fields[3])
	}

	if len(fields) >= 6 && fields[5] != "" {
		t, err := parseShellTime(fields[5])
		if err != nil {
			return data.Point{}, linePoint, err
		}
		p.Time = t
	}

	return p, linePoint, nil
}
