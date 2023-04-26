package data

import (
	"fmt"
	"log"
	"reflect"
)

// NodeEdgeChildren is used to pass a tree node structure into the
// decoder
type NodeEdgeChildren struct {
	NodeEdge NodeEdge
	Children []NodeEdgeChildren
}

type GroupedPoints struct {
	Keyed    bool
	IndexMax int
	Points   []Point
}

var reflectValueT = reflect.TypeOf(reflect.Value{})

func setValueFromPointGroup(g GroupedPoints, v reflect.Value) error {
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
			setVal(p, v.Index(int(p.Index)))
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
			setVal(p, vPtr.Elem())
			v.SetMapIndex(reflect.ValueOf(p.Key), vPtr.Elem())
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
			setVal(values[key], v.Field(i))
		}
	default:
		if len(g.Points) > 1 {
			log.Println("Decode warning, decoded multiple points to scalar")
		}
		for _, p := range g.Points {
			setVal(p, v)
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

	for _, p := range input.NodeEdge.Points {
		g, ok := pointGroups[p.Type]
		if !ok {
			g = GroupedPoints{}
		}
		if p.Key != "" {
			g.Keyed = true
		}
		if index := int(p.Index); index > g.IndexMax {
			g.IndexMax = index
		}
		g.Points = append(g.Points, p)
		pointGroups[p.Type] = g
	}
	for _, p := range input.NodeEdge.EdgePoints {
		g, ok := edgePointGroups[p.Type]
		if !ok {
			g = GroupedPoints{}
		}
		if p.Key != "" {
			g.Keyed = true
		}

		if index := int(p.Index); index > g.IndexMax {
			g.IndexMax = index
		}
		g.Points = append(g.Points, p)
		edgePointGroups[p.Type] = g
	}
	for _, c := range input.Children {
		childGroups[c.NodeEdge.Type] = append(childGroups[c.NodeEdge.Type], c)
	}

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		if pt := sf.Tag.Get("point"); pt != "" {
			g := pointGroups[pt]
			// Write points into struct field
			err := setValueFromPointGroup(g, vOut.Field(i))
			if err != nil {
				log.Printf("decode error for type %v: %v", pt, err)
			}
		} else if et := sf.Tag.Get("edgepoint"); et != "" {
			g := edgePointGroups[et]
			// Write points into struct field
			err := setValueFromPointGroup(g, vOut.Field(i))
			if err != nil {
				log.Printf("decode error for type %v: %v", et, err)
			}
		} else if nt := sf.Tag.Get("node"); nt != "" {
			// Ensure field is a settable string
			if nt == "id" {
				v := vOut.Field(i)
				if !v.CanSet() {
					log.Printf("decode error for id: cannot set")
					continue
				}
				if v.Kind() != reflect.String {
					log.Printf("decode error for id: not a string")
					continue
				}
				v.SetString(input.NodeEdge.ID)
			} else if nt == "parent" {
				v := vOut.Field(i)
				if !v.CanSet() {
					log.Printf("decode error for parent: cannot set")
					continue
				}
				if v.Kind() != reflect.String {
					log.Printf("decode error for parent: not a string")
					continue
				}
				v.SetString(input.NodeEdge.Parent)
			}
		} else if ct := sf.Tag.Get("child"); ct != "" {
			g := childGroups[ct]
			// Ensure field is a settable slice
			v := vOut.Field(i)
			t := v.Type()
			if t.Kind() != reflect.Slice {
				log.Printf("decode error for child %v: not a slice", ct)
			}
			if !v.CanSet() {
				log.Printf("decode error for child %v: cannot set", ct)
			}

			// Initialize slice
			v.Set(reflect.MakeSlice(t, len(g), len(g)))
			for i, child := range childGroups[ct] {
				err := Decode(child, v.Index(i))
				if err != nil {
					log.Printf("decode error for child %v: %v", ct, err)
				}
			}
		}
	}

	return nil
}
