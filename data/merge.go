package data

import (
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

// MergePoints takes points and updates fields in a type
// that have matching point tags. See [Decode] for an example type.
func MergePoints(points []Point, output interface{}) error {
	vOut := reflect.ValueOf(output).Elem()
	tOut := reflect.TypeOf(output).Elem()

	pointValues := make(map[string]reflect.Value)

	for i := 0; i < tOut.NumField(); i++ {
		sf := tOut.Field(i)
		if pt := sf.Tag.Get("point"); pt != "" {
			pointValues[pt] = vOut.Field(i)
		}
	}

	for _, p := range points {
		v, ok := pointValues[p.Type]
		if ok {
			setVal(p, v)
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
