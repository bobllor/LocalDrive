package diskwriter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bobllor/cloud-project/src/utils"
)

const (
	FilePerm   = 0o744
	FolderPerm = 0o777
	FileFlags  = os.O_WRONLY | os.O_CREATE
)

// NewDiskReadWriter creates a new DiskReadWriter for performing operations
// on the disk, including reading and writing.
func NewDiskReadWriter(chunkSize int, deps *utils.Deps) *DiskReadWriter {
	return &DiskReadWriter{
		chunkSize: chunkSize,
		deps:      deps,
	}
}

type DiskReadWriter struct {
	// chunkSize is the size of the chunk used to write to the disk.
	chunkSize int

	deps *utils.Deps
}

// Exists checks if the given path exists in the root path of DiskReadWriter.
func (d *DiskReadWriter) Exists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		d.deps.Log.Warnf("Failed to stat file for existence: %v | Path: %s", err, path)
		return false, err
	}
	if s == nil {
		return false, nil
	}

	return true, nil
}

// WriteToDisk writes the bytes to the disk path.
//
// It will return a FileInfo of the file if successful,
// or an error if one occurs.
//
// If an error occurs, the file will not be written to disk and the process
// will need to be started from the beginning.
func (d *DiskReadWriter) WriteToDisk(path string, data []byte) (os.FileInfo, error) {
	dataLen := len(data)
	if dataLen == 0 {
		return nil, fmt.Errorf("cannot write empty data (got %d length for bytes)", dataLen)
	}

	err := os.MkdirAll(filepath.Dir(path), FolderPerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create folders for %s: %v", path, err)
	}
	f, err := os.OpenFile(path, FileFlags, FilePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	d.deps.Log.Infof("Writing %d byte(s) to disk", len(data))

	currSize := 0
	// handles if chunkSize is larger than the given data size
	chunk := min(dataLen, d.chunkSize)

	debugCounter := 0

	for currSize < dataLen {
		if chunk > dataLen {
			chunk = dataLen
		}
		byteChunk := data[currSize:chunk]

		n, err := f.Write(byteChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to write to disk: %v", err)
		}

		currSize += n
		chunk += d.chunkSize
		debugCounter += 1
	}

	d.deps.Log.Debugf("Looped data %d time(s)", debugCounter)
	d.deps.Log.Infof("Wrote %d byte(s)", currSize)

	info, err := os.Stat(path)
	if err != nil {
		d.deps.Log.Criticalf("Failed to stat file after writing: %v | Path: %s", err, path)
		return nil, fmt.Errorf("failed to create file: %v", err)
	}

	return info, nil
}
