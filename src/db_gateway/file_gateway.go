package dbgateway

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/sqlquery"
	"github.com/bobllor/cloud-project/src/user"
	"github.com/bobllor/cloud-project/src/utils"
)

var (
	FileDoesNotExistErr = errors.New("given file ID does not exist")
	// Any errors related to SQL, such as building or querying.
	SqlErr = errors.New("an error occurred on the database server")
	// ServerErr that represents an error on the server.
	ServerErr     = errors.New("an error occured on the server")
	UniqueFileErr = errors.New("file already exists, file must be unique")
	NoFileArgsErr = errors.New("no files given")
)

// NewFileGateway creates a new FileGateway for database related options.
func NewFileGateway(database *sql.DB, deps *utils.Deps) *FileGateway {
	f := &FileGateway{
		database:       database,
		fileFieldCount: file.ColumnSize,
		deps:           deps,
	}

	return f
}

type FileGateway struct {
	database       *sql.DB
	fileFieldCount int
	deps           *utils.Deps
}

// GetAllFiles returns a FileResponse slice of all File rows belonging to the file owner.
//
// If an error occurs then it will return an error, and abort
// the scanning process if it is occurring.
func (f *FileGateway) GetAllFiles(fileOwnerID string) ([]file.FileResponse, error) {
	query, args, err := sqlquery.Select(
		file.TableName,
		file.ColumnFileName, file.ColumnFileType,
		file.ColumnFileID, file.ColumnFileExtension,
		file.ColumnParentID, file.ColumnFileSize,
		file.ColumnModifiedOn, file.ColumnDeletedOn,
		file.ColumnUploadInProgress,
	).Where().Equal(file.ColumnFileOwnerID, fileOwnerID).Build()
	if err != nil {
		f.deps.Log.Criticalf("Failed to build query: %v | Query: %s | Args: %d", err, query, len(args))
		return nil, SqlErr
	}

	rows, err := f.database.Query(query, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute query: %v | Query: %s", err, query)
		return nil, SqlErr
	}

	var files []file.FileResponse
	err = SelectRows(rows, &files)
	if err != nil {
		f.deps.Log.Criticalf("Failed to retrieve data from query: %v | Query: %s", err, query)
		return nil, SqlErr
	}

	return files, nil
}

// GetFile retrieves a single file based on the given file ID. It will return the
// full File metadata.
//
// If the file does not exist, it will return nil. This must be handled.
func (f *FileGateway) GetFile(fileOwnerId string, fileId string) (*file.File, error) {
	q, args, err := sqlquery.Select(
		file.TableName,
	).Where().Equal(file.ColumnFileOwnerID, fileOwnerId).And().Equal(file.ColumnFileID, fileId).Build()
	if err != nil {
		f.deps.Log.Criticalf("Failed to build query to update: %v | Query: %s | Args: %d", err, q, len(args))
		return nil, SqlErr
	}

	rows, err := f.database.Query(q, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute query: %v | Query: %s", err, q)
		return nil, SqlErr
	}

	fr, err := f.getFiles(rows)
	if err != nil {
		f.deps.Log.Criticalf("Failed to retrieve data from query: %v | Query: %s", err, q)
		return nil, SqlErr
	}
	if len(fr) == 0 {
		return nil, nil
	}

	return &fr[0], nil
}

// UpdateFile updates a single File's column based on its file ID and the file owner.
//
// Upon update, the modified time will also be updated to the time it was called.
func (f *FileGateway) UpdateFile(fileOwnerID string, fileId, column string, arg any) error {
	now := time.Now().UTC()
	query, args, err := sqlquery.Update(file.TableName, file.ColumnModifiedOn, column).Args(now, arg).
		Where().Equal(file.ColumnFileOwnerID, fileOwnerID).
		And().Equal(file.ColumnFileID, fileId).Build()
	if err != nil {
		f.deps.Log.Criticalf("Failed to build query: %v", err)
		return SqlErr
	}

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute query: %v | Query: %s", err, query)
		return SqlErr
	}

	f.deps.Log.Infof("Updated file %s", fileId)
	logResultRows(f.deps.Log, res)

	return nil
}

// AddFile adds slice of File structs to the File database.
//
// A special duplicate error can be returned.
// A duplicate error can occur if the following are true for a File:
//   - File.Type is "file"
//   - An existing entry with the same File.Name, File.Extension, and File.ParentID
//
// A special files length of 0 error can also be returned.
//
// Otherwise generic errors are returned for client usage. If one file fails then all
// given files will fail.
//
// This does not write the files to the disk, it is strictly used for metadata purposes.
func (f *FileGateway) AddFile(files ...file.File) error {
	if len(files) == 0 {
		return NoFileArgsErr
	}

	query, args, err := sqlquery.InsertInto(
		file.TableName,
		file.ColumnFileOwnerID,
		file.ColumnFileName,
		file.ColumnFileType,
		file.ColumnFileID,
		file.ColumnFileExtension,
		file.ColumnParentID,
		file.ColumnFilePath,
		file.ColumnFileSize,
		file.ColumnModifiedOn,
		file.ColumnDeletedOn,
		file.ColumnUploadInProgress,
		file.ColumnUniqueHash,
	).Args(file.FlattenFile(files...)...).Build()
	if err != nil {
		f.deps.Log.Criticalf("Failed to build ADD FILE INSERT INTO query: %v", err)
		return SqlErr
	}

	// duplicate errors can occur here, the original err has to be returned and handled
	res, err := execQuery(f.database, query, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to insert into %s: %v | Query: %s", file.TableName, err, query)
		return err
	}

	f.deps.Log.Infof("Successfully added %d files", len(files))
	logResultRows(f.deps.Log, res)

	return nil
}

// RenameFile renames a file to a new file name. This requires the fileId
// in order to rename.
//
// This will also update the modified date time.
func (f *FileGateway) RenameFile(accountId, fileId, newFileName string) error {
	q, args, err := sqlquery.Update(file.TableName, file.ColumnFileName, file.ColumnModifiedOn).
		Args(newFileName, time.Now().UTC()).Where().Equal(file.ColumnFileID, fileId).
		And().Equal(file.ColumnFileOwnerID, accountId).Build()
	if err != nil {
		return logSqlBuildError(f.deps.Log, err, q, args)
	}

	res, err := execQuery(f.database, q, args...)
	if err != nil {
		return logQueryError(f.deps.Log, err, q)
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// UpdateModifiedFiles updates the modified date column to the current time.
func (f *FileGateway) UpdateModifiedFiles(fileOwnerID string, fileIDs ...string) error {
	now := time.Now().UTC()
	query, args, err := sqlquery.Update(file.TableName, file.ColumnModifiedOn).Args(now).
		Where().In(file.ColumnFileID, utils.ConvertToAny(fileIDs)...).
		And().Equal(file.ColumnFileOwnerID, fileOwnerID).Build()
	if err != nil {
		return logSqlBuildError(f.deps.Log, err, query, args)
	}

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		return logQueryError(f.deps.Log, err, query)
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// DeleteFile sets a slice of file IDs to be marked for deletion.
//
// This does not delete the files immediately but marks the deletion date 15 days from today.
func (f *FileGateway) DeleteFiles(fileOwnerID string, fileIDs ...string) error {
	if len(fileIDs) == 0 {
		f.deps.Log.Critical("Failed to delete files, got file IDs length 0")
		return ServerErr
	}

	const DAYS_UNTIL_DELETE = 15

	deleteTime := time.Now().AddDate(0, 0, DAYS_UNTIL_DELETE).UTC()
	query, args, err := sqlquery.Update(file.TableName, file.ColumnDeletedOn).Args(deleteTime).
		Where().Equal(file.ColumnFileOwnerID, fileOwnerID).
		And().In(file.ColumnFileID, utils.ConvertToAny(fileIDs)...).Build()
	if err != nil {
		return logSqlBuildError(f.deps.Log, err, query, args)
	}

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute query: %v | Query: %s", err, query)
		return SqlErr
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// RestoreFiles sets a file IDs that are unmark files that were marked for deletion.
func (f *FileGateway) RestoreFiles(fileOwnerID string, fileIDs ...string) error {
	query, args, err := sqlquery.Update(file.TableName, file.ColumnDeletedOn).Args(nil).
		Where().Equal(file.ColumnFileOwnerID, fileOwnerID).
		And().In(file.ColumnFileID, utils.ConvertToAny(fileIDs)...).Build()
	if err != nil {
		return logSqlBuildError(f.deps.Log, err, query, args)
	}

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		return logQueryError(f.deps.Log, err, query)
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// GetFilesByAccountIdAndParentId retrieves the files of a given folder ID.
//
// If the given parent folder ID does not exist, it will return a 404 and an error.
func (f *FileGateway) GetFilesByAccountIdAndParentId(accountId string, parentFolderID string) ([]file.File, error) {
	if parentFolderID != "" {
		validID, err := f.validateFileExists(accountId, parentFolderID)
		if err != nil {
			f.deps.Log.Criticalf("Failed to validate file (database error): %v", err)
			return nil, err
		}

		if !validID {
			f.deps.Log.Infof("Parent ID %s does not have an existing entry", parentFolderID)
			return nil, FileDoesNotExistErr
		}
	}

	args := []any{accountId, parentFolderID}

	query := fmt.Sprintf(`
		SELECT f.*
		FROM %s f 
		JOIN %s 
			ON u.%s = f.%s 
		WHERE u.%s = ? AND f.%s = ?
		`,
		file.TableName,
		fmt.Sprintf("%s u", user.TableName),
		user.ColumnAccountID,
		file.ColumnFileOwnerID,
		user.ColumnAccountID,
		file.ColumnParentID,
	)

	rows, err := f.database.Query(query, args...)
	if err != nil {
		return nil, logQueryError(f.deps.Log, err, query)
	}

	files, err := f.getFiles(rows)
	if err != nil {
		f.deps.Log.Criticalf("Failed to retrieve (SELECT) files: %v", err)
		return nil, SqlErr
	}

	return files, nil
}

// validateFileExists checks if the folder ID has the correct formatting and
// a database query if it exists in the table.
// It requires the account ID in order to check for the existence of the folder to the correct
// owner.
//
// If no errors occur it will return true for validation. Any failures will return false.
// If an error occurs, it will return an error.
func (f *FileGateway) validateFileExists(accountID string, fileID string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s f 
		JOIN %s u
			ON u.%s = f.%s 
		WHERE u.%s = ? AND f.%s = ?
		`,
		file.TableName,
		user.TableName,
		user.ColumnAccountID,
		file.ColumnFileOwnerID,
		user.ColumnAccountID,
		file.ColumnFileID,
	)

	rows, err := f.database.Query(query, accountID, fileID)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute database query: %v | query: %s", err, query)
		return false, SqlErr
	}

	type Counter struct{ Count int }
	var counter Counter
	err = SelectRow(rows, &counter)
	if err != nil {
		f.deps.Log.Criticalf("Failed to retrieve rows with query: %v", err)
		return false, SqlErr
	}
	f.deps.Log.Debugf("Rows found: %v", counter)

	if counter.Count == 0 {
		return false, nil
	}

	return true, nil
}

// getFiles is a helper function used to scan and return
// a slice of Files.
//
// sql.Rows will automatically be closed at the end of function.
func (f *FileGateway) getFiles(rows *sql.Rows) ([]file.File, error) {
	files := []file.File{}

	for rows.Next() {
		f := file.File{}

		scanErr := rows.Scan(
			&f.OwnerID,
			&f.Name,
			&f.Type,
			&f.FileID,
			&f.Extension,
			&f.ParentID,
			&f.Path,
			&f.Size,
			&f.ModifiedOn,
			&f.DeletedOn,
			&f.UploadInProgress,
			&f.UniqueHash,
		)

		if scanErr != nil {
			return nil, scanErr
		}

		files = append(files, f)
	}

	return files, nil
}
