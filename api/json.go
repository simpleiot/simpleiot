package api

import (
	"encoding/json"
	"io"
)

func decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}
