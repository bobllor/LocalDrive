package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// ConvertToAny converts a slice to an []any. This is only used for
// for query arguments.
func ConvertToAny[S ~[]T, T comparable](v S) []any {
	conv := []any{}

	for _, e := range v {
		conv = append(conv, e)
	}

	return conv
}

// FormatTime formats the given time to the format YYYY-MM-DD HH:MM:SS
// as a string.
func FormatTime(date time.Time) string {
	return date.Format(time.DateTime)
}

// GetFiles retrieves all files in a given path.
// It will return a map with the key being the lowercased file name
// and the value being the absolute path of the file.
//
// This does not recursively search through directories.
func GetFiles(root string) (map[string]string, error) {
	files := make(map[string]string)

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, ent := range entries {
		files[strings.ToLower(ent.Name())] = filepath.Join(root, ent.Name())
	}

	return files, nil
}

// StructToAny flattens a struct into an any slice, maintaining
// the field order from top to bottom in the slice.
//
// Nil values will return an empty slice.
//
// If the given value is not a struct, then it will return an empty slice.
func StructToAny(s any) []any {
	out := []any{}
	v := reflect.ValueOf(s)
	v = getReflectValue(v)

	if v.Kind() != reflect.Struct {
		return out
	}

	for i := range v.NumField() {
		fieldValue := v.Field(i).Interface()

		out = append(out, fieldValue)
	}

	return out
}

// getReflectValue gets the value of s. It is a recursive function
// that will return the first non-pointer value.
func getReflectValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer {
		return getReflectValue(v.Elem())
	}

	return v
}
