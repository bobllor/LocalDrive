package utils

import (
	"testing"
	"time"

	"github.com/bobllor/assert"
)

func TestStructToAny(t *testing.T) {
	type testStruct struct {
		FirstName string
		LastName  string
		BirthDate *time.Time
	}

	t.Run("Normal", func(t *testing.T) {
		s := testStruct{
			FirstName: "John",
			LastName:  "Doe",
		}

		val := StructToAny(s)
		assert.Equal(t, len(val), 3)
	})

	t.Run("Pointers", func(t *testing.T) {
		structs := []*testStruct{
			{
				FirstName: "John",
				LastName:  "Doe",
			},
			nil,
		}

		for _, s := range structs {
			val := StructToAny(s)
			if s != nil {
				assert.Equal(t, len(val), 3)
			} else {
				assert.Equal(t, len(val), 0)
			}
		}
	})
}
