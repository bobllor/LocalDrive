package utils

import (
	"encoding/json"
	"io"
)

// Decode decodes a given io.Reader into a given type T's default value.
//
// If an error occurs or v does not decode into T, then it will return its
// nil value.
func Decode[T any](v io.Reader) (T, error) {
	var zero T
	err := json.NewDecoder(v).Decode(&zero)
	if err != nil {
		return zero, err
	}

	return zero, nil
}
