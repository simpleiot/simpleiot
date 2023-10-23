package data

import (
	"fmt"
	"reflect"
	"strconv"
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
	k := v.Kind()

	if k == reflect.Pointer {
		if v.IsNil() {
			p.Tombstone = 1
			return p, nil
		}
		v = v.Elem()
		k = v.Kind()
	}
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
				return points, fmt.Errorf("%v of %w", k, err)
			}
			p.Key = strconv.Itoa(i)
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
	case reflect.Pointer:
		// We support pointers to primitives and structs
		// If the pointer is nil, all generated points will have a tombstone set
		if !v.IsNil() {
			return appendPointsFromValue(points, pointType, v.Elem())
		}
		switch k := t.Elem().Kind(); k {
		case reflect.Struct:
			// Generate a tombstone point for all struct fields
			numField := t.Elem().NumField()
			if numField > maxStructureSize {
				return points, fmt.Errorf(
					"%v size of %v exceeds maximum of %v",
					k, numField, maxStructureSize,
				)
			}
			for i := 0; i < numField; i++ {
				sf := t.Elem().Field(i)
				key := sf.Tag.Get("point")
				if key == "" {
					key = sf.Tag.Get("edgepoint")
				}
				if key == "" {
					key = ToCamelCase(sf.Name)
				}
				p := Point{
					Type:      pointType,
					Key:       key,
					Tombstone: 1,
				}
				points = append(points, p)
			}
		case reflect.Bool,
			reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Float32,
			reflect.Float64,
			reflect.String:
			points = append(points, Point{Type: pointType, Tombstone: 1})
		default:
			return points, fmt.Errorf("unsupported pointer type: %v", k)
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
func Encode(in any) (NodeEdge, error) {
	inV, inT, inK := reflectValue(in)
	nodeType := ToCamelCase(inT.Name())
	ret := NodeEdge{Type: nodeType}

	if inK != reflect.Struct {
		return ret, fmt.Errorf("error decoding to %v; must be a struct", inK)
	}

	var err error
	for i := 0; i < inT.NumField(); i++ {
		sf := inT.Field(i)
		if pt := sf.Tag.Get("point"); pt != "" {
			ret.Points, err = appendPointsFromValue(
				ret.Points, pt, inV.Field(i),
			)
			if err != nil {
				return ret, err
			}
		} else if et := sf.Tag.Get("edgepoint"); et != "" {
			ret.EdgePoints, err = appendPointsFromValue(
				ret.EdgePoints, et, inV.Field(i),
			)
			if err != nil {
				return ret, err
			}
		} else if nt := sf.Tag.Get("node"); nt != "" &&
			sf.Type.Kind() == reflect.String {

			if nt == "id" {
				ret.ID = inV.Field(i).String()
			} else if nt == "parent" {
				ret.Parent = inV.Field(i).String()
			}
		}
	}

	return ret, nil
}

// DiffPoints compares a before and after struct and generates the set of Points
// that represent their differences.
func DiffPoints[T any](before, after T) (Points, error) {
	bV, t, k := reflectValue(before)
	aV, _, _ := reflectValue(after)

	// Check to ensure this is a struct
	if k != reflect.Struct {
		return nil, fmt.Errorf("error decoding to %v; must be a struct", k)
	}

	points := Points{}
	for i, numFields := 0, t.NumField(); i < numFields; i++ {
		// Determine point type from struct tag
		structTag := t.Field(i).Tag
		pointType := structTag.Get("point")
		if pointType == "" {
			continue
		}

		bFieldV := bV.Field(i)
		aFieldV := aV.Field(i)

		// Handle special case of pointer to a struct
		if bFieldV.Kind() == reflect.Pointer &&
			bFieldV.Type().Elem().Kind() == reflect.Struct {
			// If new pointer is nil, set all fields to tombstone, else
			// proceed
			if bFieldV.IsNil() && aFieldV.IsNil() {
				// do nothing
				continue
			} else if aFieldV.IsNil() {
				// Generate a tombstone point for all struct fields
				t := bFieldV.Type().Elem()
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
						key = ToCamelCase(sf.Name)
					}
					p := Point{
						Type:      pointType,
						Key:       key,
						Tombstone: 1,
					}
					points.Add(p)
				}
				continue
			} else if bFieldV.IsNil() {
				var err error
				points, err = appendPointsFromValue(
					points,
					pointType,
					aFieldV.Elem(),
				)
				if err != nil {
					return points, err
				}
				continue
			}

			bFieldV = bFieldV.Elem()
			aFieldV = aFieldV.Elem()
		}

		switch bFieldV.Kind() {
		case reflect.Array, reflect.Slice:
			if aFieldV.Len() > maxStructureSize {
				return points, fmt.Errorf(
					"%v length of %v exceeds maximum of %v",
					k, aFieldV.Len(), maxStructureSize,
				)
			}
			i, aFieldLen, bFieldLen := 0, aFieldV.Len(), bFieldV.Len()
			for ; i < aFieldLen; i++ {
				if i >= bFieldLen || !aFieldV.Index(i).Equal(bFieldV.Index(i)) {
					// Add / update point
					p, err := pointFromPrimitive(pointType, aFieldV.Index(i))
					if err != nil {
						return points, fmt.Errorf("%v of %w", k, err)
					}
					p.Key = strconv.Itoa(i)
					points.Add(p)
				}
			}
			for i = bFieldLen - 1; i >= aFieldLen; i-- {
				// Create tombstone point
				points.Add(Point{
					Type:      pointType,
					Key:       strconv.Itoa(i),
					Tombstone: 1,
				})
			}
		case reflect.Map:
			// Points support maps with string keys
			if keyK := bFieldV.Type().Key().Kind(); keyK != reflect.String {
				return points, fmt.Errorf("unsupported type: map keyed by %v", keyK)
			}
			if aFieldV.Len() > maxStructureSize {
				return points, fmt.Errorf(
					"%v length of %v exceeds maximum of %v",
					k, aFieldV.Len(), maxStructureSize,
				)
			}
			// Populate keysToDelete with all keys from `before` map
			keysToDelete := make(map[string]bool)
			iter := bFieldV.MapRange()
			for iter.Next() {
				keysToDelete[iter.Key().String()] = true
			}
			// Now iterate over `after` map
			iter = aFieldV.MapRange()
			for iter.Next() {
				mKey, mVal := iter.Key(), iter.Value()
				if !mVal.Equal(bFieldV.MapIndex(mKey)) {
					// Add / update key
					p, err := pointFromPrimitive(pointType, mVal)
					if err != nil {
						return points, fmt.Errorf("map contains %w", err)
					}
					p.Key = mKey.String()
					points.Add(p)
				}
				delete(keysToDelete, mKey.String())
			}
			for key := range keysToDelete {
				// Create tombstone point
				points.Add(Point{
					Type:      pointType,
					Key:       key,
					Tombstone: 1,
				})
			}
		case reflect.Struct:
			// Points support "flat" structs, and they are treated like maps
			// Key name is taken from struct "point" tag or from the field name
			t := bFieldV.Type()
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
					key = ToCamelCase(sf.Name)
				}
				if !bFieldV.Field(i).Equal(aFieldV.Field(i)) {
					// Update key
					p, err := pointFromPrimitive(pointType, aFieldV.Field(i))
					if err != nil {
						return points, fmt.Errorf("struct contains %w", err)
					}
					p.Key = key
					points.Add(p)
				}
			}
		default:
			if !bFieldV.Equal(aFieldV) {
				// Update point
				p, err := pointFromPrimitive(pointType, aFieldV)
				if err != nil {
					return points, err
				}
				points.Add(p)
			}
		}
	}
	return points, nil
}
