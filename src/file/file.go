package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// FileColumnSize is the amount of columns used for the Files table.
	// It is equal to the public fields of the [File] struct.
	ColumnSize          int    = 10
	TableName           string = "File"
	ColumnFileOwnerID   string = "AccountID" // ColumnFileOwnerID is the column name for the file's owner account ID.
	ColumnFileName      string = "FileName"
	ColumnFileType      string = "FileType"
	ColumnFileID        string = "FileID"
	ColumnFileExtension string = "Extension"
	ColumnParentID      string = "ParentID"
	ColumnFilePath      string = "FilePath"
	ColumnFileSize      string = "FileSize"
	ColumnModifiedOn    string = "ModifiedOn"
	ColumnDeletedOn     string = "DeletedOn"
)

type FileType string

const (
	FileTypeDir  FileType = "dir"
	FileTypeFile FileType = "file"
)

type File struct {
	// OwnerID is the ID of the owner of the file.
	OwnerID string `json:"accountID"`

	// Name is the name of the file. This includes the extension
	// of the file.
	Name string `json:"fileName"`

	// Type is the file type. This is either a "directory" or
	// a "file".
	Type FileType `json:"fileType"`

	// FileID is a unique ID assigned to the file.
	FileID string `json:"fileID"`

	// Extension is the extension of the file type.
	Extension string `json:"extension"`

	// ParentID is the parent's unique ID that the file resides in.
	// This can be nil, meaning it resides in the root folder.
	ParentID *string `json:"parentID"`

	// Path is the absolute path to the file on the disk. This is intended
	// for the backend use only and should not be sent to the frontend.
	Path string

	// Size is the size of the file.
	Size int64 `json:"fileSize"`

	// ModifiedOn is the most recent time the file has been modified. This
	// will be the most recent time of change or when it was first created.
	// This must be in UTC.
	ModifiedOn time.Time `json:"modifedOn"`

	// DeletedOn is the time when the file is set to be deleted. The acutal
	// deletion occurs after a certain amount of time has passed
	// since the marked deletion time. This value can be nil.
	// This must be in UTC.
	DeletedOn *time.Time `json:"deletedOn"`
}

// FileResponse is the struct representing a File object
// from the backend. It is the same struct as File, excluding
// the field FilePath and OwnerID.
type FileResponse struct {
	Name       string     `json:"fileName"`
	Type       FileType   `json:"fileType"`
	FileID     string     `json:"fileID"`
	Extension  string     `json:"extension"`
	ParentID   *string    `json:"parentID"`
	Size       int64      `json:"fileSize"`
	ModifiedOn time.Time  `json:"modifedOn"`
	DeletedOn  *time.Time `json:"deletedOn"`
}

// Read returns a File slice for all files found in root.
// An error will be returned if there is an issue while reading root.
//
// This is only intended for local disk access, and is not intended to be
// used for adding files into the database outside of some situations.
// Adding into the database is delegated to the API call.
func Read(root string) ([]File, error) {
	fs, err := walk(root)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// FlattenFiles flattens the a slice of File structs to prepare for use in
// a query.
func FlattenFile(files ...File) []any {
	out := []any{}

	appendFunc := func(v any) {
		out = append(out, v)
	}

	for _, file := range files {
		appendFunc(file.OwnerID)
		appendFunc(file.Name)
		appendFunc(file.Type)
		appendFunc(file.FileID)
		appendFunc(file.Extension)
		appendFunc(file.ParentID)
		appendFunc(file.Path)
		appendFunc(file.Size)
		appendFunc(file.ModifiedOn)
		appendFunc(file.DeletedOn)
	}

	return out
}

// ToFileResponses converts a File slice into a FileResponse
// slice.
//
// This is intended to be used for client responses to refrain
// sending confidential fields.
func ToFileResponses(files ...File) []FileResponse {
	responses := []FileResponse{}

	for _, file := range files {
		responses = append(responses, *file.ToFileResponse())
	}

	return responses
}

// StringClean returns a clean string representation of File excluding secrets.
// This does not include the account ID.
//
// The formatting is: "<key>=<val>;..."
func (f *File) StringClean() string {
	data := []string{}
	write := func(key string, val any) {
		data = append(data, fmt.Sprintf("%s=%v", key, val))
	}

	write(ColumnFileName, f.Name)
	write(ColumnFileType, f.Type)
	write(ColumnFileID, f.FileID)
	write(ColumnFileExtension, f.Extension)
	write(ColumnParentID, f.ParentID)
	write(ColumnFileSize, f.Size)
	write(ColumnModifiedOn, f.ModifiedOn.UTC().String())

	var deletedOn any
	if f.DeletedOn != nil {
		deletedOn = f.DeletedOn.UTC().String()
	}
	write(ColumnDeletedOn, deletedOn)

	return strings.Join(data, ";")
}

// ToFileResponse converts the File struct into a
// FileResponse struct.
//
// This is intended to be used for client responses to refrain
// sending confidential fields.
func (f *File) ToFileResponse() *FileResponse {
	return &FileResponse{
		Name:       f.Name,
		Type:       f.Type,
		FileID:     f.FileID,
		Extension:  f.Extension,
		ParentID:   f.ParentID,
		Size:       f.Size,
		ModifiedOn: f.ModifiedOn,
		DeletedOn:  f.DeletedOn,
	}
}

// walk is used to traverse root and return a File slice for
// all the files in root.
//
// If any error occurs during the file reading, it will abort the process
// and return an error.
func walk(root string) ([]File, error) {
	fs := []File{}

	folderIDMap := map[string]*string{
		root: nil,
	}

	// folder name of root is the account ID
	// this only is applicable to local files
	accountID := filepath.Base(root)

	walkFunc := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		id := uuid.NewString()

		// skipping root
		if p != root {
			fileType := FileTypeFile

			var parentID *string
			parent := filepath.Dir(p)
			if info.IsDir() {
				fileType = FileTypeDir
				_, ok := folderIDMap[p]
				if !ok {
					folderIDMap[p] = &id
				}
			}

			pID, ok := folderIDMap[parent]
			if ok {
				parentID = pID
			}

			f := File{
				Name:       info.Name(),
				Type:       fileType,
				Size:       info.Size(),
				Extension:  filepath.Ext(p),
				Path:       p,
				FileID:     id,
				ParentID:   parentID,
				ModifiedOn: info.ModTime().UTC(),
				OwnerID:    accountID,
			}

			fs = append(fs, f)
		}

		return nil
	}

	err := filepath.Walk(root, walkFunc)
	if err != nil {
		return nil, err
	}

	return fs, nil
}
