package file

import (
	"testing"

	"github.com/bobllor/assert"
)

func TestFailRead(t *testing.T) {
	_, err := Read("dir/does/not/exist")
	assert.NotNil(t, err)
}

func TestFlattenFile(t *testing.T) {
	f := File{}

	flattened := FlattenFile(f)

	assert.Equal(t, ColumnSize, len(flattened))
}
