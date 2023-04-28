package data

import (
	"errors"
	"fmt"
	"reflect"
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
func MergePoints(id string, points []Point, output interface{}) error {
	// FIXME: this is not the most efficient algorithm as it recurses into
	// all child arrays, maybe optimize later

	var vOut *reflect.Value
	var tOut reflect.Type

	if reflect.TypeOf(output).String() == "*reflect.Value" {
		outputV, ok := output.(*reflect.Value)
		if !ok {
			return errors.New("Error converting interface")
		}

		vOut = outputV
		tOut = outputV.Type()
	} else {
		vOutX := reflect.ValueOf(output).Elem()
		vOut = &vOutX
		tOut = reflect.TypeOf(output).Elem()
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
		for _, p := range points {
			v, ok := pointValues[p.Type]
			if ok {
				if err := setVal(p, v); err != nil {
					return fmt.Errorf(
						"merge error for point type %v: %w",
						p.Type, err,
					)
				}
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
					return fmt.Errorf("Error merging child points: %w", err)
				}
			}
		}
	}

	return nil
}

// MergeEdgePoints takes edge points and updates a type that
// matching edgepoint tags. See [Decode] for an example type.
func MergeEdgePoints(id, parent string, points []Point, output interface{}) error {
	vOut := reflect.Indirect(reflect.ValueOf(output))
	tOut := vOut.Type()

	if tOut == reflectValueT {
		// `output` was a reflect.Value or *reflect.Value
		vOut = vOut.Interface().(reflect.Value)
		tOut = vOut.Type()
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
		for _, p := range points {
			v, ok := edgeValues[p.Type]
			if ok {
				if err := setVal(p, v); err != nil {
					return fmt.Errorf(
						"merge error for edge type %v: %w",
						p.Type, err,
					)
				}
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
					return fmt.Errorf("merge error for child edge points: %w", err)
				}
			}
		}
	}

	return nil
}
