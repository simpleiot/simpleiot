package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/simpleiot/simpleiot/data"
)

// MqttSub maps one MQTT topic into points on this node. It is a child of an
// Mqtt node.
type MqttSub struct {
	ID          string  `node:"id"`
	Parent      string  `node:"parent"`
	Description string  `point:"description"`
	Topic       string  `point:"topic"`
	Path        string  `point:"path"`
	Units       string  `point:"units"`
	Scale       float64 `point:"scale"`
	Offset      float64 `point:"offset"`
	Disabled    bool    `point:"disabled"`
	Error       string  `point:"error"`
	// Tags are ordinary tag points, carried here so the client can tell
	// whether the topic tag it maintains is already up to date.
	Tags map[string]string `point:"tag"`
}

// scaleFactor returns the multiplier to apply to numeric values. A scale of
// zero would read every message as zero, which is never what is wanted, so an
// unset scale is treated as one.
func (s MqttSub) scaleFactor() float64 {
	if s.Scale == 0 {
		return 1
	}

	return s.Scale
}

// points converts one MQTT payload into the points to publish on this node.
//
// With a path set, the value at that location becomes a single point. With a
// blank path, a bare number or string becomes a single point and an object
// becomes one point per field, keyed by the field name -- the "node per
// device" shape described in docs/user/plc.md.
func (s MqttSub) points(payload []byte) (data.Points, error) {
	var v any

	if err := json.Unmarshal(payload, &v); err != nil {
		return nil, fmt.Errorf("payload is not JSON: %w", err)
	}

	if s.Path != "" {
		found, err := jsonPath(v, s.Path)
		if err != nil {
			return nil, err
		}

		p, err := s.point("", found)
		if err != nil {
			return nil, err
		}

		return data.Points{p}, nil
	}

	obj, ok := v.(map[string]any)
	if !ok {
		p, err := s.point("", v)
		if err != nil {
			return nil, err
		}

		return data.Points{p}, nil
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	pts := make(data.Points, 0, len(keys))

	for _, k := range keys {
		p, err := s.point(data.SubjectSafeToken(k), obj[k])
		if err != nil {
			return nil, fmt.Errorf("field %v: %w", k, err)
		}

		pts = append(pts, p)
	}

	return pts, nil
}

// point converts one JSON value into a point. Numbers are scaled and offset;
// strings become text points, and booleans become 1 or 0 so a rule can compare
// them the way it compares any other on/off value.
func (s MqttSub) point(key string, v any) (data.Point, error) {
	switch t := v.(type) {
	case float64:
		return data.NewPointFloat(data.PointTypeValue, key, t*s.scaleFactor()+s.Offset), nil
	case string:
		return data.NewPointString(data.PointTypeValue, key, t), nil
	case bool:
		v := 0.0
		if t {
			v = 1
		}
		return data.NewPointFloat(data.PointTypeValue, key, v), nil
	case nil:
		return data.Point{}, fmt.Errorf("value is null")
	default:
		return data.Point{}, fmt.Errorf("value is a %T, which does not map to a point", v)
	}
}

// jsonPath returns the value at path within a decoded JSON document. The path
// is written in the dot notation JSON payload documentation generally uses --
// `$.a.b[0]`, with the leading `$` optional.
func jsonPath(v any, path string) (any, error) {
	steps, err := parseJSONPath(path)
	if err != nil {
		return nil, err
	}

	cur := v

	for i, step := range steps {
		switch {
		case step.field != "":
			obj, ok := cur.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%v: expected an object, found %T", pathSoFar(steps, i), cur)
			}

			cur, ok = obj[step.field]
			if !ok {
				return nil, fmt.Errorf("%v: not present in the payload", pathSoFar(steps, i))
			}

		default:
			arr, ok := cur.([]any)
			if !ok {
				return nil, fmt.Errorf("%v: expected an array, found %T", pathSoFar(steps, i), cur)
			}

			if step.index >= len(arr) {
				return nil, fmt.Errorf("%v: the array holds %v elements", pathSoFar(steps, i), len(arr))
			}

			cur = arr[step.index]
		}
	}

	return cur, nil
}

// jsonPathStep is one step of a path: a field name, or an array index when
// field is blank.
type jsonPathStep struct {
	field string
	index int
}

func pathSoFar(steps []jsonPathStep, upto int) string {
	var b strings.Builder
	b.WriteByte('$')

	for _, s := range steps[:upto+1] {
		if s.field != "" {
			b.WriteByte('.')
			b.WriteString(s.field)
		} else {
			b.WriteString("[" + strconv.Itoa(s.index) + "]")
		}
	}

	return b.String()
}

func parseJSONPath(path string) ([]jsonPathStep, error) {
	rest := strings.TrimPrefix(path, "$")

	var steps []jsonPathStep

	for rest != "" {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			end := strings.IndexAny(rest, ".[")
			if end < 0 {
				end = len(rest)
			}

			if end == 0 {
				return nil, fmt.Errorf("path %q has an empty field name", path)
			}

			steps = append(steps, jsonPathStep{field: rest[:end]})
			rest = rest[end:]

		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return nil, fmt.Errorf("path %q is missing a closing ]", path)
			}

			i, err := strconv.Atoi(rest[1:end])
			if err != nil || i < 0 {
				return nil, fmt.Errorf("path %q has an invalid array index %q", path, rest[1:end])
			}

			steps = append(steps, jsonPathStep{index: i})
			rest = rest[end+1:]

		default:
			// a path may start with a bare field name, as in "a.b"
			end := strings.IndexAny(rest, ".[")
			if end < 0 {
				end = len(rest)
			}

			steps = append(steps, jsonPathStep{field: rest[:end]})
			rest = rest[end:]
		}
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("path %q selects nothing", path)
	}

	return steps, nil
}
