package data

import (
	"errors"
	"fmt"
	"reflect"
)

// NodeEdgeChildren is used to pass a tree node structure into the
// decoder
type NodeEdgeChildren struct {
	NodeEdge NodeEdge
	Children []NodeEdgeChildren
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
	var vOut reflect.Value
	var tOut reflect.Type

	if reflect.TypeOf(output).String() == "*reflect.Value" {
		outputV, ok := output.(*reflect.Value)
		if !ok {
			return errors.New("Error converting interface")
		}

		vOut = outputV.Elem()
		tOut = outputV.Type().Elem()
	} else {
		vOut = reflect.ValueOf(output).Elem()
		tOut = reflect.TypeOf(output).Elem()
	}

	pointValues := make(map[string]reflect.Value)
	edgeValues := make(map[string]reflect.Value)
	childValues := make(map[string]reflect.Value)

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		if pt := sf.Tag.Get("point"); pt != "" {
			pointValues[pt] = vOut.Field(i)
		} else if et := sf.Tag.Get("edgepoint"); et != "" {
			edgeValues[et] = vOut.Field(i)
		} else if nt := sf.Tag.Get("node"); nt != "" {
			if nt == "id" {
				vOut.Field(i).SetString(input.NodeEdge.ID)
			} else if nt == "parent" {
				vOut.Field(i).SetString(input.NodeEdge.Parent)
			}
		} else if ct := sf.Tag.Get("child"); ct != "" {
			childValues[ct] = vOut.Field(i)
		}
	}

	for _, p := range input.NodeEdge.Points {
		v, ok := pointValues[p.Type]
		if ok {
			setVal(p, v)
		}
	}

	for _, p := range input.NodeEdge.EdgePoints {
		v, ok := edgeValues[p.Type]
		if ok {
			setVal(p, v)
		}
	}

	for _, c := range input.Children {
		for k, v := range childValues {
			// v is an array
			if c.NodeEdge.Type == k {
				// get an empty value of the type of the array
				cOut := reflect.New(v.Type().Elem())

				err := Decode(c, &cOut)
				if err != nil {
					return fmt.Errorf("Error decoding child: %v", err)
				}

				// append the new value to the child array
				childValues[k].Set(reflect.Append(v, cOut.Elem()))
			}
		}
	}

	return nil
}
