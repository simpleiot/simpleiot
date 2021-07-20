package data

import "errors"

// ErrDocumentNotFound is returned in APIs if document is not found
var ErrDocumentNotFound = errors.New("document not found")
