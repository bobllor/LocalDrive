package utils

import (
	"testing"
	"time"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/tests"
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

func TestHashString(t *testing.T) {
	s := HashString(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileName, "txt")

	// obtained from online
	baseHash := "f3bf6020372579f86aadbb42a6416de73463cef330a84c6a1cf493eea411a2dd"

	assert.Equal(t, len(s), 64)
	assert.Equal(t, s, baseHash)
}

func TestRemoveDuplicates(t *testing.T) {
	vs := []string{"hello", "there", "hello", "no", "yes", "yes"}
	baseLen := len(vs)

	vs = RemoveDuplicates(vs)
	newLen := len(vs)

	assert.NotEqual(t, baseLen, newLen)
}
