package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/tests"
)

func TestRead(t *testing.T) {
	dir := t.TempDir()

	_, err := tests.CreateFiles(dir)
	assert.Nil(t, err)

	files, err := Read(dir)
	assert.Nil(t, err)
	assert.NotEqual(t, len(files), 0)

	for _, file := range files {
		_, err := os.Stat(file.Path)
		assert.Nil(t, err)
	}
}

func TestReadParentIdFolders(t *testing.T) {
	dir := t.TempDir()

	_, err := tests.CreateFiles(dir)
	assert.Nil(t, err)

	files, err := Read(dir)
	assert.Nil(t, err)
	assert.NotEqual(t, len(files), 0)

	// a map of the parents [Dir(File.FilePath)]-[File] of
	// the files
	parentMap := map[string]File{}
	// flat map of [File.FileID]-[File] for all
	// the files
	fileIdMap := map[string]File{}

	for _, file := range files {
		_, err := os.Stat(file.Path)
		assert.Nil(t, err)

		_, ok := parentMap[file.Path]
		if !ok {
			parentMap[filepath.Dir(file.Path)] = file
		}

		_, ok = fileIdMap[file.FileID]
		if !ok {
			fileIdMap[file.FileID] = file
		}
	}

	for parentPath, file := range parentMap {
		if file.ParentID != nil {
			// if this fails, then the parent doesn't exist
			// in the flat file map
			idParentFile, ok := fileIdMap[*file.ParentID]
			assert.True(t, ok)

			// checks if the flat parent file is a directory
			assert.Equal(t, idParentFile.Type, FileTypeDir)
			// checks if the flat parent path is not root
			assert.NotEqual(t, idParentFile.Path, dir)
			// checks if the flat parent path is the same as the parentMap key
			assert.Equal(t, idParentFile.Path, parentPath)
		} else {
			// nil ID means the parent is root
			assert.Equal(t, parentPath, dir)
			assert.Equal(t, filepath.Dir(file.Path), dir)
		}
	}
}

func TestFailRead(t *testing.T) {
	_, err := Read("dir/does/not/exist")
	assert.NotNil(t, err)
}

func TestFlattenFile(t *testing.T) {
	f := File{}

	flattened := FlattenFile(f)

	assert.Equal(t, ColumnSize, len(flattened))
}
