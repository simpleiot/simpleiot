package data

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strconv"
)

// NodeEdgeChildren is used to pass a tree node structure into the
// decoder
type NodeEdgeChildren struct {
	NodeEdge NodeEdge
	Children []NodeEdgeChildren
}

// reflectValueT is the `reflect.Type` for a `reflect.Value`
var reflectValueT = reflect.TypeOf(reflect.Value{})

// GroupedPoints are Points grouped by their `Point.Type`. While scanning
// through the list of points, we also keep track of whether or not the
// points are keyed with positive integer values (for decoding into arrays)
type GroupedPoints struct {
	// KeyNotIndex is set to a Point's `Key` field if it *cannot* be parsed as a
	// positive integer
	// Note: If `Key` is empty string (""), it is treated as "0"
	KeyNotIndex string
	// KeyMaxInt is the largest `Point.Key` value in Points
	KeyMaxInt int
	// Points is the list of Points for this group
	Points []Point

	// TODO: We can also simultaneously implement sorting floating point keys
	// KeysNumeric is set if and only if each Point's `Key` field is numeric
	// and can be parsed as a float64
	// KeysNumeric bool
	// KeyMaxFloat is the largest `Point.Key` value in Points
	// KeyMaxFloat float64
}

// SetValue populates v with the Points in the group
func (g GroupedPoints) SetValue(v reflect.Value) error {
	t := v.Type()
	switch k := t.Kind(); k {
	case reflect.Array, reflect.Slice:
		// Ensure all keys are array indexes
		if g.KeyNotIndex != "" {
			return fmt.Errorf(
				"Point.Key %v is not a valid index",
				g.KeyNotIndex,
			)
		}
		// Check array bounds
		if g.KeyMaxInt > maxStructureSize {
			return fmt.Errorf(
				"Point.Key %v exceeds %v",
				g.KeyMaxInt, maxStructureSize,
			)
		}
		if k == reflect.Array && g.KeyMaxInt > t.Len()-1 {
			return fmt.Errorf(
				"Point.Key %v exceeds array size %v",
				g.KeyMaxInt, t.Len(),
			)
		}
		// Expand slice if needed
		if k == reflect.Slice && g.KeyMaxInt > v.Len()-1 {
			if !v.CanSet() {
				return fmt.Errorf("cannot set value %v", v)
			}
			newV := reflect.MakeSlice(t, g.KeyMaxInt+1, g.KeyMaxInt+1)
			reflect.Copy(newV, v)
			v.Set(newV)
		}
		// Set array / slice values
		deletedIndexes := []int{}
		for _, p := range g.Points {
			// Note: array / slice values are set directly on the indexed Value
			index, _ := strconv.Atoi(p.Key)
			if p.Tombstone%2 == 1 {
				deletedIndexes = append(deletedIndexes, index)
				// Ignore this deleted value if it won't fit in the slice anyway
				// Note: KeyMaxInt is not set for points with Tombstone set, so
				// index could still be out of range.
				if index >= v.Len() {
					continue
				}
			}
			// Finally, set the value in the slice
			err := setVal(p, v.Index(index))
			if err != nil {
				return err
			}
		}
		// We can now trim the underlying slice to remove trailing values that
		// were deleted in this decode. Note: this does not guarantee that
		// slices are always trimmed properly because values can be deleted
		// across multiple decode cycles.
		slices.Sort(deletedIndexes)
		lastIndex := v.Len() - 1
		for i := len(deletedIndexes) - 1; i >= 0; i-- {
			if deletedIndexes[i] != lastIndex {
				break
			}
			lastIndex--
		}
		v.Set(v.Slice(0, lastIndex+1))
	case reflect.Map:
		// Ensure map is keyed by string
		if keyK := t.Key().Kind(); keyK != reflect.String {
			return fmt.Errorf("cannot set map keyed by %v", keyK)
		}
		if len(g.Points) > maxStructureSize {
			return fmt.Errorf(
				"number of points %v exceeds maximum of %v for a map",
				len(g.Points), maxStructureSize,
			)
		}
		// Ensure points are keyed
		// Note: No longer relevant, as all points as keyed now
		// if !g.Keyed {
		// 	return fmt.Errorf("points missing Key")
		// }

		// Ensure map is initialized
		if v.IsNil() {
			if !v.CanSet() {
				return fmt.Errorf("cannot set value %v", v)
			}
			v.Set(reflect.MakeMapWithSize(t, len(g.Points)))
		}
		// Set map values
		for _, p := range g.Points {
			// Enforce valid Key value
			key := p.Key
			if key == "" {
				key = "0"
			}
			if p.Tombstone%2 == 1 {
				// We want to delete the map entry if Tombstone is set
				v.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
			} else {
				// Create and set a new map value
				// Note: map values must be set on newly created Values
				// because (unlike arrays / slices) any value returned by
				// `MapIndex` is not settable
				newV := reflect.New(t.Elem()).Elem()
				err := setVal(p, newV)
				if err != nil {
					return err
				}
				v.SetMapIndex(reflect.ValueOf(key), newV)
			}
		}
	case reflect.Struct:
		// Create map of Points
		values := make(map[string]Point, len(g.Points))
		for _, p := range g.Points {
			values[p.Key] = p
		}
		// Write points to struct
		for numField, i := v.NumField(), 0; i < numField; i++ {
			sf := t.Field(i)
			key := sf.Tag.Get("point")
			if key == "" {
				key = sf.Tag.Get("edgepoint")
			}
			if key == "" {
				key = ToCamelCase(sf.Name)
			}
			// Ensure points are keyed
			if key == "" {
				return fmt.Errorf("point missing Key")
			}
			err := setVal(values[key], v.Field(i))
			if err != nil {
				return err
			}
		}
	default:
		if len(g.Points) > 1 {
			log.Printf(
				"Decode warning, decoded multiple points to %v:\n%v",
				// Cast to `Points` type with a `String()` method which prints
				// a trailing newline
				t, Points(g.Points),
			)
		}
		for _, p := range g.Points {
			err := setVal(p, v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Decode converts a Node to custom struct.
// output can be a struct type that contains
// node, point, and edgepoint tags as shown below.
// It is recommended that id and parent node tags
// always be included.
//
//	   type exType struct {
//		ID          string      `node:"id"`
//		Parent      string      `node:"parent"`
//		Description string      `point:"description"`
//		Count       int         `point:"count"`
//		Role        string      `edgepoint:"role"`
//		Tombstone   bool        `edgepoint:"tombstone"`
//		Conditions  []Condition `child:"condition"`
//	   }
//
// outputStruct can also be a *reflect.Value
func Decode(input NodeEdgeChildren, outputStruct interface{}) error {
	outV, outT, outK := reflectValue(outputStruct)
	if outK != reflect.Struct {
		return fmt.Errorf("error decoding to %v; must be a struct", outK)
	}
	var retErr error

	// Group points and children by type
	pointGroups := make(map[string]GroupedPoints)
	edgePointGroups := make(map[string]GroupedPoints)
	childGroups := make(map[string][]NodeEdgeChildren)

	// we first collect all points into groups by type
	// this is required in case we are decoding into a map or array
	// Note: Even points with tombstones set are processed here; later we set
	// the destination to the zero value if a tombstone is present.
	for _, p := range input.NodeEdge.Points {
		g := pointGroups[p.Type] // uses zero value if not found
		if p.Key != "" {
			index, err := strconv.Atoi(p.Key)
			if err != nil || index < 0 {
				g.KeyNotIndex = p.Key
			} else if index > g.KeyMaxInt && p.Tombstone%2 == 0 {
				// Note: Do not set `KeyMaxInt` if Tombstone is set. We don't
				// need to expand the slice in this case.
				g.KeyMaxInt = index
			}
		}
		// else p.Key is treated like "0"; no need to update `g` at all
		g.Points = append(g.Points, p)
		pointGroups[p.Type] = g
	}
	for _, p := range input.NodeEdge.EdgePoints {
		g := edgePointGroups[p.Type]
		if p.Key != "" {
			index, err := strconv.Atoi(p.Key)
			if err != nil || index < 0 {
				g.KeyNotIndex = p.Key
			} else if index > g.KeyMaxInt && p.Tombstone%2 == 0 {
				g.KeyMaxInt = index
			}
		}
		g.Points = append(g.Points, p)
		edgePointGroups[p.Type] = g
	}
	for _, c := range input.Children {
		childGroups[c.NodeEdge.Type] = append(childGroups[c.NodeEdge.Type], c)
	}

	// now process the fields in the output struct
	for i := 0; i < outT.NumField(); i++ {
		sf := outT.Field(i)
		// look at tags to determine if we have a point, edgepoint, node attribute, or child node
		if pt := sf.Tag.Get("point"); pt != "" {
			// see if we have any points for this field point type
			g, ok := pointGroups[pt]
			if ok {
				// Write points into struct field
				err := g.SetValue(outV.Field(i))
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for type %v: %w", pt, err,
					))
				}
			}
		} else if et := sf.Tag.Get("edgepoint"); et != "" {
			g, ok := edgePointGroups[et]
			if ok {
				// Write points into struct field
				err := g.SetValue(outV.Field(i))
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for type %v: %w", et, err,
					))
				}
			}
		} else if nt := sf.Tag.Get("node"); nt != "" {
			// Set ID or Parent field where appropriate
			if nt == "id" && input.NodeEdge.ID != "" {
				v := outV.Field(i)
				if !v.CanSet() {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for id: cannot set",
					))
					continue
				}
				if v.Kind() != reflect.String {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for id: not a string",
					))
					continue
				}
				v.SetString(input.NodeEdge.ID)
			} else if nt == "parent" && input.NodeEdge.Parent != "" {
				v := outV.Field(i)
				if !v.CanSet() {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for parent: cannot set",
					))
					continue
				}
				if v.Kind() != reflect.String {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for parent: not a string",
					))
					continue
				}
				v.SetString(input.NodeEdge.Parent)
			}
		} else if ct := sf.Tag.Get("child"); ct != "" {
			g, ok := childGroups[ct]
			if ok {
				// Ensure field is a settable slice
				v := outV.Field(i)
				t := v.Type()
				if t.Kind() != reflect.Slice {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for child %v: not a slice", ct,
					))
					continue
				}
				if !v.CanSet() {
					retErr = errors.Join(retErr, fmt.Errorf(
						"decode error for child %v: cannot set", ct,
					))
					continue
				}

				// Initialize slice
				v.Set(reflect.MakeSlice(t, len(g), len(g)))
				for i, child := range g {
					err := Decode(child, v.Index(i))
					if err != nil {
						retErr = errors.Join(retErr, fmt.Errorf(
							"decode error for child %v: %w", ct, err,
						))
					}
				}
			}
		}
	}

	return retErr
}

// setVal writes a scalar Point value / text to a reflect.Value
// Supports boolean, integer, floating point, and string destinations
// Writes the zero value to `v` if the Point has an odd Tombstone value
func setVal(p Point, v reflect.Value) error {
	if !v.CanSet() {
		return fmt.Errorf("cannot set value")
	}
	if p.Tombstone%2 == 1 {
		// Set to zero value
		v.Set(reflect.Zero(v.Type()))
		return nil
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
			return fmt.Errorf("uint overflow: %v", p.Value)
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

// reflectValue returns a reflect.Value from an interface
// This function dereferences `output` if it's a pointer or a reflect.Value
func reflectValue(output interface{}) (
	outV reflect.Value, outT reflect.Type, outK reflect.Kind,
) {
	outV = reflect.Indirect(reflect.ValueOf(output))
	outT = outV.Type()

	if outT == reflectValueT {
		// `output` was a reflect.Value or *reflect.Value
		outV = outV.Interface().(reflect.Value)
		outT = outV.Type()
	}

	outK = outT.Kind()
	return
}
