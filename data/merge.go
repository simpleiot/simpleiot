package data

import (
	"errors"
	"fmt"
	"log"
	"reflect"
)

func setVal(p Point, v reflect.Value) {
	switch v.Type().Kind() {
	case reflect.String:
		v.SetString(p.Text)
	case reflect.Int:
		v.SetInt(int64(p.Value))
	case reflect.Float64, reflect.Float32:
		v.SetFloat(p.Value)
	case reflect.Bool:
		v.SetBool(FloatToBool(p.Value))
	default:
		log.Println("setVal failed, did not match any type: ", v.Type().Kind())
	}
}

var count = 0

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
				setVal(p, v)
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
					return fmt.Errorf("Error merging child points: %v", err)
				}
			}
		}
	}

	return nil
}

// MergeEdgePoints takes edge points and updates a type that
// matching edgepoint tags. See [Decode] for an example type.
func MergeEdgePoints(points []Point, output interface{}) error {
	vOut := reflect.ValueOf(output).Elem()
	tOut := reflect.TypeOf(output).Elem()

	edgeValues := make(map[string]reflect.Value)

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		if et := sf.Tag.Get("edgepoint"); et != "" {
			edgeValues[et] = vOut.Field(i)
		}
	}

	for _, p := range points {
		v, ok := edgeValues[p.Type]
		if ok {
			setVal(p, v)
		}
	}

	return nil
}
