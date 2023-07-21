package data

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// setVal writes a scalar Point value / text to a reflect.Value
func setVal(p Point, v reflect.Value) error {
	if !v.CanSet() {
		return fmt.Errorf("cannot set value")
	}
	switch k := v.Kind(); k {
	case reflect.Bool:
		v.SetBool(FloatToBool(p.Value))
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:

		if v.OverflowInt(int64(p.Value)) {
			return fmt.Errorf("int overflow: %v", p.Value)
		}
		v.SetInt(int64(p.Value))
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:

		if p.Value < 0 || v.OverflowUint(uint64(p.Value)) {
			return fmt.Errorf("int overflow: %v", p.Value)
		}
		v.SetUint(uint64(p.Value))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(p.Value)
	case reflect.String:
		v.SetString(p.Text)
	default:
		return fmt.Errorf("unsupported type: %v", k)
	}
	return nil
}

// MergePoints takes points and updates fields in a type
// that have matching point tags. See [Decode] for an example type.
// When deleting points from arrays, the point key (index) is ignored
// and the last entry from the array is removed. Normally, it is recommended
// to send all points for an array when doing complex modifications to
// an array.
func MergePoints(id string, points []Point, output interface{}) error {
	// TODO: this is not the most efficient algorithm as it recurses into
	// all child arrays looking for a struct id

	var retErr error
	vOut := reflect.Indirect(reflect.ValueOf(output))
	tOut := vOut.Type()

	if tOut == reflectValueT {
		// `output` was a reflect.Value or *reflect.Value
		vOut = vOut.Interface().(reflect.Value)
		tOut = vOut.Type()
	}

	if k := tOut.Kind(); k != reflect.Struct {
		return fmt.Errorf("Error decoding to %v; must be a struct", k)
	}

	pointGroups := make(map[string]GroupedPoints)

	for _, p := range points {
		g, ok := pointGroups[p.Type]
		if !ok {
			g = GroupedPoints{}
		}
		if p.Key != "" {
			g.Keyed = true
		}

		index, _ := strconv.Atoi(p.Key)
		if index > g.IndexMax {
			g.IndexMax = index
		}
		g.Points = append(g.Points, p)
		pointGroups[p.Type] = g
	}

	pointValues := make(map[string]reflect.Value)
	childValues := make(map[string]reflect.Value)

	structID := ""

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		if pt := sf.Tag.Get("point"); pt != "" {
			pointValues[pt] = vOut.Field(i)
		} else if nt := sf.Tag.Get("node"); nt != "" {
			if nt == "id" {
				structID = vOut.Field(i).String()
			}
		} else if ct := sf.Tag.Get("child"); ct != "" {
			childValues[ct] = vOut.Field(i)
		}
	}

	if structID == id {
		for k, v := range pointValues {
			g, ok := pointGroups[k]
			if !ok {
				continue
			}

			// Write points into struct field
			err := g.SetValue(v)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("decode error for type %v: %w", k, err))
			}
		}
	} else if len(childValues) > 0 {
		// try children
		for _, children := range childValues {
			// v is an array, iterate through child array
			for i := 0; i < children.Len(); i++ {
				v := children.Index(i)
				err := MergePoints(id, points, &v)
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("Error merging child points: %w", err))
				}
			}
		}
	}

	return retErr
}

// MergeEdgePoints takes edge points and updates a type that
// matching edgepoint tags. See [Decode] for an example type.
func MergeEdgePoints(id, parent string, points []Point, output interface{}) error {

	var retErr error
	vOut := reflect.Indirect(reflect.ValueOf(output))
	tOut := vOut.Type()

	if tOut == reflectValueT {
		// `output` was a reflect.Value or *reflect.Value
		vOut = vOut.Interface().(reflect.Value)
		tOut = vOut.Type()
	}

	if k := tOut.Kind(); k != reflect.Struct {
		return fmt.Errorf("Error decoding to %v; must be a struct", k)
	}

	edgeGroups := make(map[string]GroupedPoints)

	for _, p := range points {
		g, ok := edgeGroups[p.Type]
		if !ok {
			g = GroupedPoints{}
		}
		if p.Key != "" {
			g.Keyed = true
		}
		index, _ := strconv.Atoi(p.Key)
		if index > g.IndexMax {
			g.IndexMax = index
		}
		g.Points = append(g.Points, p)
		edgeGroups[p.Type] = g
	}

	edgeValues := make(map[string]reflect.Value)
	childValues := make(map[string]reflect.Value)

	structID := ""
	structParent := ""

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		if et := sf.Tag.Get("edgepoint"); et != "" {
			edgeValues[et] = vOut.Field(i)
		} else if nt := sf.Tag.Get("node"); nt != "" {
			if nt == "id" {
				structID = vOut.Field(i).String()
			} else if nt == "parent" {
				structParent = vOut.Field(i).String()
			}
		} else if ct := sf.Tag.Get("child"); ct != "" {
			childValues[ct] = vOut.Field(i)
		}
	}

	if structID == id && structParent == parent {
		for k, v := range edgeValues {
			g, ok := edgeGroups[k]
			// TODO: may be an optimization to check ok and not call SetValue if value
			// in map does not exist, rather than processing the zero value
			if !ok {
				continue
			}

			// Write points into struct field
			err := g.SetValue(v)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("decode error for type %v: %w", k, err))
			}
		}
	} else if len(childValues) > 0 {
		// try children
		for _, children := range childValues {
			// v is an array, iterate through child array
			for i := 0; i < children.Len(); i++ {
				v := children.Index(i)
				err := MergeEdgePoints(id, parent, points, &v)
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("merge error for child edge points: %w", err))
				}
			}
		}
	}

	return retErr
}
