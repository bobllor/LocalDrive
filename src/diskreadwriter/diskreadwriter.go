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

// NewDiskWriter creates a new DiskWriter for writing
// data to the disk.
func NewDiskReadWriter(chunkSize int, deps *utils.Deps) *DiskWriter {
	return &DiskWriter{
		chunkSize: chunkSize,
		deps:      deps,
	}
}

type DiskWriter struct {
	chunkSize int
	deps      *utils.Deps
}

// WriteToDisk writes the bytes to the disk path.
//
// It will return a FileInfo of the file if successful,
// or an error if one occurs.
//
// If an error occurs, the file will not be written to disk and the process
// will need to be started from the beginning.
func (dw *DiskWriter) WriteToDisk(path string, data []byte) (os.FileInfo, error) {
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

	dw.deps.Log.Infof("Writing %d byte(s) to disk", len(data))

	currSize := 0
	// handles if chunkSize is larger than the given data size
	chunk := min(dataLen, dw.chunkSize)

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
		chunk += dw.chunkSize
		debugCounter += 1
	}

	dw.deps.Log.Debugf("Looped data %d time(s)", debugCounter)
	dw.deps.Log.Infof("Wrote %d byte(s)", currSize)

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}

	return info, nil
}
