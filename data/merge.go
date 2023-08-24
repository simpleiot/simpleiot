package data

import (
	"fmt"
	"reflect"
)

// FindNodeInStruct recursively scans the `outputStruct` for a struct
// with a field having a `node:"id"` tag and whose value matches `nodeID`.
// If `parentID` is provided, the struct must also have a field with a
// `node:"parent"` tag whose value matches `parentID`. If such a struct is
// found, the struct is returned as a reflect.Value; otherwise, an invalid
// reflect.Value is returned whose IsValid method returns false.
func FindNodeInStruct(outputStruct interface{}, nodeID string, parentID string) reflect.Value {
	// If `nodeID` is not provided, we abort immediately
	if nodeID == "" {
		return reflect.Value{}
	}

	outV, outT, outK := reflectValue(outputStruct)
	if outK != reflect.Struct {
		return reflect.Value{}
	}

	// Scan struct fields
	outID := ""
	outParentID := ""
	childValues := make(map[string]reflect.Value)

	for i := 0; i < outT.NumField(); i++ {
		sf := outT.Field(i)
		if nt := sf.Tag.Get("node"); nt != "" {
			if nt == "id" {
				outID = outV.Field(i).String()
			} else if nt == "parent" {
				outParentID = outV.Field(i).String()
			}
		} else if ct := sf.Tag.Get("child"); ct != "" &&
			sf.Type.Kind() == reflect.Slice {
			childValues[ct] = outV.Field(i)
		}
	}

	if parentID == "" {
		if outID == nodeID {
			return outV // found it
		}
	} else if outID == nodeID && outParentID == parentID {
		return outV // found it
	}

	// `outV` does not match; scan all children recursively
	for _, c := range childValues {
		// Note: `c` was already checked to ensure it's a slice
		for i, length := 0, c.Len(); i < length; i++ {
			childVal := FindNodeInStruct(c.Index(i), nodeID, parentID)
			if childVal.IsValid() {
				return childVal // found it
			}
		}
	}

	// Not found; return invalid value
	return reflect.Value{}
}

// MergePoints takes points and updates fields in a type
// that have matching point tags. See [Decode] for an example type.
// When deleting points from arrays, the point key (index) is ignored
// and the last entry from the array is removed. Normally, it is recommended
// to send all points for an array when doing complex modifications to
// an array.
func MergePoints(id string, points []Point, outputStruct interface{}) error {
	outV := FindNodeInStruct(outputStruct, id, "")
	if !outV.IsValid() {
		return fmt.Errorf(
			"no matching struct with a `node:\"id\"` field matching %v", id,
		)
	}

	ne := NodeEdge{
		ID:     id,
		Points: points,
	}
	return Decode(NodeEdgeChildren{NodeEdge: ne}, outV)
}

// MergeEdgePoints takes edge points and updates a type that
// matching edgepoint tags. See [Decode] for an example type.
func MergeEdgePoints(id, parent string, points []Point, outputStruct interface{}) error {
	outV := FindNodeInStruct(outputStruct, id, parent)
	if !outV.IsValid() {
		return fmt.Errorf(
			"no matching struct with a `node:\"id\"` field matching %v", id,
		)
	}

	ne := NodeEdge{
		ID:         id,
		Parent:     parent,
		EdgePoints: points,
	}
	return Decode(NodeEdgeChildren{NodeEdge: ne}, outV)
}
