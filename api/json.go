package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func decode(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

func encode(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

// decodeError answers a request whose body could not be decoded: 413 when
// it was over the size limit, 400 otherwise.
func decodeError(res http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		http.Error(res, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(res, err.Error(), http.StatusBadRequest)
}
