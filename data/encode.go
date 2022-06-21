package data

import (
	"fmt"
	"reflect"
)

// Encode is used to convert a user struct to
// a node.
func Encode(in interface{}) (NodeEdge, error) {
	vIn := reflect.ValueOf(in)
	tIn := reflect.TypeOf(in)

	ret := NodeEdge{Type: tIn.Name()}

	valToPoint := func(t string, v reflect.Value) (Point, error) {
		k := v.Type().Kind()
		switch k {
		case reflect.String:
			return Point{Type: t, Text: v.String()}, nil
		case reflect.Int:
			return Point{Type: t, Value: float64(v.Int())}, nil
		case reflect.Float64:
			return Point{Type: t, Value: v.Float()}, nil
		case reflect.Float32:
			f := float32(v.Float())
			return Point{Type: t, Value: float64(f)}, nil
		case reflect.Bool:
			return Point{Type: t, Value: BoolToFloat(v.Bool())}, nil
		default:
			return Point{}, fmt.Errorf("Unhandled type: %v", k)
		}
	}

	for i := 0; i < tIn.NumField(); i++ {
		sf := tIn.Field(i)
		pt := sf.Tag.Get("point")
		if pt != "" {
			p, err := valToPoint(pt, vIn.Field(i))
			if err != nil {
				return ret, err
			}
			ret.Points = append(ret.Points, p)
		} else {
			et := sf.Tag.Get("edgepoint")
			if et != "" {
				p, err := valToPoint(et, vIn.Field(i))
				if err != nil {
					return ret, err
				}
				ret.EdgePoints = append(ret.EdgePoints, p)
			}
		}
	}

	return ret, nil
}
