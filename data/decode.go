package data

import (
	"errors"
	"fmt"
	"log"
	"reflect"
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

// GroupedPoints are Points grouped by the `Point.Type`. While scanning
// through the list of points, we also keep track of whether or not the
// points have an Index or Key
type GroupedPoints struct {
	// Keyed if and only if a Point's `Key` field is populated
	Keyed bool
	// IndexMax is the largest `Point.Index` value in Points
	IndexMax int
	// Points is the list of Points for this group
	Points []Point
}

// SetValue populates v with the Points in the group
func (g GroupedPoints) SetValue(v reflect.Value) error {
	t := v.Type()
	switch k := t.Kind(); k {
	case reflect.Array, reflect.Slice:
		// Check array bounds
		if g.IndexMax > maxStructureSize {
			return fmt.Errorf(
				"Point.Index %v exceeds %v",
				g.IndexMax, maxStructureSize,
			)
		}
		if k == reflect.Array && g.IndexMax > t.Len()-1 {
			return fmt.Errorf(
				"Point.Index %v exceeds array size %v",
				g.IndexMax, t.Len(),
			)
		}
		// Expand slice if needed
		if k == reflect.Slice && g.IndexMax > v.Len()-1 {
			if !v.CanSet() {
				return fmt.Errorf("cannot set value %v", v)
			}
			newV := reflect.MakeSlice(t, g.IndexMax+1, g.IndexMax+1)
			reflect.Copy(newV, v)
			v.Set(newV)
		}
		// Set array / slice values
		for _, p := range g.Points {
			// Note: array / slice values are set directly on the indexed Value
			index, _ := strconv.Atoi(p.Key)
			if p.Tombstone == 1 {
				// assume the entire array is written, so if there are
				// tombstone points sent, simply remove the last entry
				// in the array as a point to remove previous entries
				// may be sent first.
				newLen := v.Len() - 1
				newSlice := reflect.MakeSlice(v.Type(), newLen, newLen)
				for i := 0; i < newLen; i++ {
					newSlice.Index(i).Set(v.Index(i))
				}
				v.Set(newSlice)
			} else {
				err := setVal(p, v.Index(index))
				if err != nil {
					return err
				}
			}
		}
	case reflect.Map:
		// Ensure map is keyed by string
		if keyK := t.Key().Kind(); keyK != reflect.String {
			return fmt.Errorf("cannot set map keyed by %v", keyK)
		}
		if len(g.Points) > maxStructureSize {
			return fmt.Errorf(
				"size of %v exceeds maximum of %v",
				len(g.Points), maxStructureSize,
			)
		}
		// Ensure points are keyed
		if !g.Keyed {
			return fmt.Errorf("points missing Key")
		}
		// Ensure map is initialized
		if v.IsNil() {
			if !v.CanSet() {
				return fmt.Errorf("cannot set value %v", v)
			}
			v.Set(reflect.MakeMapWithSize(t, len(g.Points)))
		}
		// Set map values
		for _, p := range g.Points {
			// Create and set a new map value
			// Note: map values must be set on newly created Values
			vPtr := reflect.New(t.Elem())
			err := setVal(p, vPtr.Elem())
			if err != nil {
				return err
			}
			v.SetMapIndex(reflect.ValueOf(p.Key), vPtr.Elem())
		}
	case reflect.Struct:
		// Ensure points are keyed
		if !g.Keyed {
			return fmt.Errorf("points missing Key")
		}
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
			err := setVal(values[key], v.Field(i))
			if err != nil {
				return err
			}
		}
	default:
		if len(g.Points) > 1 {
			log.Printf("Decode warning, decoded multiple points to %v:\n%v", t, Points(g.Points))
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
// output can also be a *reflect.Value
func Decode(input NodeEdgeChildren, output interface{}) error {
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

	// Group points and children by type
	pointGroups := make(map[string]GroupedPoints)
	edgePointGroups := make(map[string]GroupedPoints)
	childGroups := make(map[string][]NodeEdgeChildren)

	// we first collect all points into groups by type
	// this is required in case we are decoding into a map or array
	for _, p := range input.NodeEdge.Points {
		if p.Tombstone == 1 {
			continue
		}
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
	for _, p := range input.NodeEdge.EdgePoints {
		if p.Tombstone == 1 {
			continue
		}
		g, ok := edgePointGroups[p.Type]
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
		edgePointGroups[p.Type] = g
	}
	for _, c := range input.Children {
		childGroups[c.NodeEdge.Type] = append(childGroups[c.NodeEdge.Type], c)
	}

	// now process the fields in the output struct
	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		// look at tags to determine if we have a point, edgepoint, node attribute, or child node
		if pt := sf.Tag.Get("point"); pt != "" {
			// see if we have any points for this field point type
			g, ok := pointGroups[pt]
			if ok {
				// Write points into struct field
				err := g.SetValue(vOut.Field(i))
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for type %v: %w", pt, err))
				}
			}
		} else if et := sf.Tag.Get("edgepoint"); et != "" {
			g, ok := edgePointGroups[et]
			if ok {
				// Write points into struct field
				err := g.SetValue(vOut.Field(i))
				if err != nil {
					retErr = errors.Join(err, fmt.Errorf("decode error for type %v: %w", et, err))
				}
			}
		} else if nt := sf.Tag.Get("node"); nt != "" {
			// Ensure field is a settable string
			if nt == "id" {
				v := vOut.Field(i)
				if !v.CanSet() {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for id: cannot set"))
					continue
				}
				if v.Kind() != reflect.String {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for id: not a string"))
					continue
				}
				v.SetString(input.NodeEdge.ID)
			} else if nt == "parent" {
				v := vOut.Field(i)
				if !v.CanSet() {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for parent: cannot set"))
					continue
				}
				if v.Kind() != reflect.String {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for parent: not a string"))
					continue
				}
				v.SetString(input.NodeEdge.Parent)
			}
		} else if ct := sf.Tag.Get("child"); ct != "" {
			g, ok := childGroups[ct]
			if ok {
				// Ensure field is a settable slice
				v := vOut.Field(i)
				t := v.Type()
				if t.Kind() != reflect.Slice {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for child %v: not a slice", ct))
				}
				if !v.CanSet() {
					retErr = errors.Join(retErr, fmt.Errorf("decode error for child %v: cannot set", ct))
				}

				// Initialize slice
				v.Set(reflect.MakeSlice(t, len(g), len(g)))
				for i, child := range childGroups[ct] {
					err := Decode(child, v.Index(i))
					if err != nil {
						retErr = errors.Join(retErr, fmt.Errorf("decode error for child %v: %v", ct, err))
					}
				}
			}
		}
	}

	return retErr
}
