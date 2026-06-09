package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobllor/cloud-project/src/utils"
	"github.com/google/uuid"
)

const (
	// FileColumnSize is the amount of columns used for the Files table.
	// It is equal to the public fields of the [File] struct.
	ColumnSize             int    = 12
	TableName              string = "File"
	ColumnFileOwnerID      string = "AccountID" // ColumnFileOwnerID is the column name for the file's owner account ID.
	ColumnFileName         string = "FileName"
	ColumnFileType         string = "FileType"
	ColumnFileID           string = "FileID"
	ColumnFileExtension    string = "Extension"
	ColumnParentID         string = "ParentID"
	ColumnFilePath         string = "FilePath"
	ColumnFileSize         string = "FileSize"
	ColumnModifiedOn       string = "ModifiedOn"
	ColumnDeletedOn        string = "DeletedOn"
	ColumnUploadInProgress string = "UploadInProgress"
	ColumnUniqueHash       string = "UniqueHash"
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

	// ParentID is the parent's file ID that the file resides in.
	// If it is an empty string then it is considered to be in the root folder.
	ParentID string `json:"parentID"`

	// Path is the relative path to the file on the disk. The path to the
	// storage is used with the file path.
	//
	// This is intended for the backend use only and should not be sent
	// to the frontend.
	Path string

	// Size is the size of the file.
	Size int64 `json:"fileSize"`

	// ModifiedOn is the most recent time the file has been modified. This
	// will be the most recent time of change or when it was first created.
	// This must be in UTC.
	ModifiedOn time.Time `json:"modifedOn"`

	// DeletedOn is the time when the file is set to be deleted. The actual
	// deletion occurs after a certain amount of time has passed
	// since the marked deletion time. This value can be nil.
	// This must be in UTC.
	DeletedOn *time.Time `json:"deletedOn"`

	// UploadInProgress is the status of the file if it is currently
	// in an uploading process. If it is true, then its intention is
	// to prevent the front end from showing the file.
	UploadInProgress bool `json:"uploadInProgress"`

	// UniqueHash is a SHA256 hash of the following string concatenation:
	// 	- account id + file name + file extension + parent ID
	//
	// This is used to keep files unique in the database if they
	// have the same parent ID under an account.
	UniqueHash string `json:"uniqueHash"`
}

// FileResponse is the struct representing a File object
// from the backend. It is the same struct as File, excluding
// the field FilePath, OwnerID, and UniqueHash.
type FileResponse struct {
	Name             string     `json:"fileName"`
	Type             FileType   `json:"fileType"`
	FileID           string     `json:"fileID"`
	Extension        string     `json:"extension"`
	ParentID         string     `json:"parentID"`
	Size             int64      `json:"fileSize"`
	ModifiedOn       time.Time  `json:"modifedOn"`
	DeletedOn        *time.Time `json:"deletedOn"`
	UploadInProgress bool       `json:"uploadInProgress"`
}

// NewFile creates a new File with the file ID, modified date,
// and deleted date having their values handled in the constructor.
//
// The following fields will automatically be generated and their side effects:
//   - Extension: automatically gets stripped of leading periods
//   - UniqueHash: concatenation of account id, file name, file extension, and parent ID (in order)
//   - Path: <account ID>/<file ID>
//
// The path will be an empty path if the file type is a directory.
func NewFile(accountId string,
	fileName string,
	fileType FileType,
	fileExt string,
	fileSize int64,
	parentId string,
	uploadInProg bool) File {
	id := uuid.NewString()

	fileExt = strings.TrimPrefix(fileExt, ".")
	hash := utils.HashString(accountId, fileName, fileExt, parentId)

	path := ""
	if fileType != FileTypeDir {
		path = fmt.Sprintf("%s/%s", accountId, id)
	}

	return File{
		OwnerID:          accountId,
		Name:             fileName,
		Type:             fileType,
		FileID:           id,
		Extension:        fileExt,
		ParentID:         parentId,
		Path:             path,
		Size:             fileSize,
		ModifiedOn:       time.Now().UTC(),
		DeletedOn:        nil,
		UploadInProgress: uploadInProg,
		UniqueHash:       hash,
	}
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
		appendFunc(file.UploadInProgress)
		appendFunc(file.UniqueHash)
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
	write(ColumnUploadInProgress, f.UploadInProgress)
	write(ColumnUniqueHash, f.UniqueHash)

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
		Name:             f.Name,
		Type:             f.Type,
		FileID:           f.FileID,
		Extension:        f.Extension,
		ParentID:         f.ParentID,
		Size:             f.Size,
		ModifiedOn:       f.ModifiedOn,
		DeletedOn:        f.DeletedOn,
		UploadInProgress: f.UploadInProgress,
	}
}

// walk is used to traverse root and return a File slice for
// all the files in root.
//
// If any error occurs during the file reading, it will abort the process
// and return an error.
func walk(root string) ([]File, error) {
	fs := []File{}

	folderIDMap := map[string]string{
		root: "",
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

			var parentID string
			parent := filepath.Dir(p)
			if info.IsDir() {
				fileType = FileTypeDir
				_, ok := folderIDMap[p]
				if !ok {
					folderIDMap[p] = id
				}
			}

			pID, ok := folderIDMap[parent]
			if ok {
				parentID = pID
			}

			f := NewFile(
				accountID,
				info.Name(),
				fileType,
				filepath.Ext(p),
				info.Size(),
				parentID,
				false,
			)
			// manually changiing these due to the way walk is parsed
			f.FileID = id
			f.Path = fmt.Sprintf("%s/%s", accountID, id)

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
