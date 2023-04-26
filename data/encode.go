package data

import (
	"fmt"
	"reflect"
	"strings"
)

// maxSafeInteger is the largest integer value that can be stored in a float64
// (i.e. Point value) without losing precision.
const maxSafeInteger = 1<<53 - 1

// maxStructureSize is the largest array / map / struct that will be converted
// to an array of Points
const maxStructureSize = 1000

func pointFromPrimitive(pointType string, v reflect.Value) (Point, error) {
	p := Point{Type: pointType}
	k := v.Type().Kind()
	switch k {
	case reflect.Bool:
		p.Value = BoolToFloat(v.Bool())
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:

		val := v.Int()
		if val > maxSafeInteger || val < -maxSafeInteger {
			return p, fmt.Errorf("float64 overflow for value: %v", val)
		}
		p.Value = float64(val)
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:

		val := v.Uint()
		if val > maxSafeInteger {
			return p, fmt.Errorf("float64 overflow for value: %v", val)
		}
		p.Value = float64(val)
	case reflect.Float32, reflect.Float64:
		p.Value = v.Float()
	case reflect.String:
		p.Text = v.String()
	default:
		return p, fmt.Errorf("unsupported type: %v", k)
	}
	return p, nil
}
func appendPointsFromValue(
	points []Point,
	pointType string,
	v reflect.Value,
) ([]Point, error) {
	t := v.Type()
	k := t.Kind()
	switch k {
	case reflect.Array, reflect.Slice:
		// Points support arrays / slices of supported primitives
		if v.Len() > maxStructureSize {
			return points, fmt.Errorf(
				"%v length of %v exceeds maximum of %v",
				k, v.Len(), maxStructureSize,
			)
		}
		for i := 0; i < v.Len(); i++ {
			p, err := pointFromPrimitive(pointType, v.Index(i))
			if err != nil {
				return points, fmt.Errorf("unsupported type: %v of %w", k, err)
			}
			p.Index = float32(i)
			points = append(points, p)
		}
	case reflect.Map:
		// Points support maps with string keys
		if keyK := t.Key().Kind(); keyK != reflect.String {
			return points, fmt.Errorf("unsupported type: map keyed by %v", keyK)
		}
		if v.Len() > maxStructureSize {
			return points, fmt.Errorf(
				"%v length of %v exceeds maximum of %v",
				k, v.Len(), maxStructureSize,
			)
		}
		iter := v.MapRange()
		for iter.Next() {
			mKey, mVal := iter.Key(), iter.Value()
			p, err := pointFromPrimitive(pointType, mVal)
			if err != nil {
				return points, fmt.Errorf("map contains %w", err)
			}
			p.Key = mKey.String()
			points = append(points, p)
		}
	case reflect.Struct:
		// Points support "flat" structs, and they are treated like maps
		// Key name is taken from struct "point" tag or from the field name
		numField := t.NumField()
		if numField > maxStructureSize {
			return points, fmt.Errorf(
				"%v size of %v exceeds maximum of %v",
				k, numField, maxStructureSize,
			)
		}
		for i := 0; i < numField; i++ {
			sf := t.Field(i)
			key := sf.Tag.Get("point")
			if key == "" {
				key = sf.Tag.Get("edgepoint")
			}
			if key == "" {
				key = ToCamelCase(sf.Name)
			}
			p, err := pointFromPrimitive(pointType, v.Field(i))
			if err != nil {
				return points, fmt.Errorf("struct contains %w", err)
			}
			p.Key = key
			points = append(points, p)
		}
	default:
		p, err := pointFromPrimitive(pointType, v)
		if err != nil {
			return points, err
		}
		points = append(points, p)
	}
	return points, nil
}

// ToCamelCase naively converts a string to camelCase. This function does
// not consider common initialisms.
func ToCamelCase(s string) string {
	// Find first lowercase letter
	lowerIndex := strings.IndexFunc(s, func(c rune) bool {
		return 'a' <= c && c <= 'z'
	})
	if lowerIndex < 0 {
		// ALLUPPERCASE
		s = strings.ToLower(s)
	} else if lowerIndex == 1 {
		// FirstLetterUppercase
		s = strings.ToLower(s[0:lowerIndex]) + s[lowerIndex:]
	} else if lowerIndex > 1 {
		// MANYLettersUppercase
		s = strings.ToLower(s[0:lowerIndex-1]) + s[lowerIndex-1:]
	}
	return s
}

// Encode is used to convert a user struct to
// a node. in must be a struct type that contains
// node, point, and edgepoint tags as shown below.
// It is recommended that id and parent node tags
// always be included.
//
//	   type exType struct {
//		ID          string  `node:"id"`
//		Parent      string  `node:"parent"`
//		Description string  `point:"description"`
//		Count       int     `point:"count"`
//		Role        string  `edgepoint:"role"`
//		Tombstone   bool    `edgepoint:"tombstone"`
//	   }
func Encode(in interface{}) (NodeEdge, error) {
	vIn := reflect.ValueOf(in)
	tIn := reflect.TypeOf(in)

	nodeType := ToCamelCase(tIn.Name())

	ret := NodeEdge{Type: nodeType}
	var err error

	for i := 0; i < tIn.NumField(); i++ {
		sf := tIn.Field(i)
		if pt := sf.Tag.Get("point"); pt != "" {
			ret.Points, err = appendPointsFromValue(
				ret.Points, pt, vIn.Field(i),
			)
			if err != nil {
				return ret, err
			}
		} else if et := sf.Tag.Get("edgepoint"); et != "" {
			ret.EdgePoints, err = appendPointsFromValue(
				ret.EdgePoints, et, vIn.Field(i),
			)
			if err != nil {
				return ret, err
			}
		} else if nt := sf.Tag.Get("node"); nt != "" &&
			sf.Type.Kind() == reflect.String {

			if nt == "id" {
				ret.ID = vIn.Field(i).String()
			} else if nt == "parent" {
				ret.Parent = vIn.Field(i).String()
			}
		}
	}

	return ret, nil
}
