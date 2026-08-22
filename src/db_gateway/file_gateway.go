package dbgateway

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
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
// It will automatically be sorted in ascending order with dir > file and in alphabetical
// order.
//
// This includes any files set to be deleted.
//
// If an error occurs then it will return an error, and abort
// the scanning process if it is occurring.
func (f *FileGateway) GetAllFiles(fileOwnerID string) ([]file.FileResponse, error) {
	query := fmt.Sprintf(
		`SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = ?
		ORDER BY %s, %s`,
		file.ColumnFileName, file.ColumnFileType,
		file.ColumnFileID, file.ColumnFileExtension,
		file.ColumnParentID, file.ColumnFileSize,
		file.ColumnModifiedOn, file.ColumnDeletedOn,
		file.ColumnUploadStatus, file.ColumnUniqueHash,
		file.TableName,
		file.ColumnFileOwnerID,
		file.ColumnFileType, file.ColumnFileName,
	)

	args := []any{fileOwnerID}

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

// GetDeletedFiles retrieves all files that are marked for deletion, or if the
// deleted column date is not null.
//
// The file response will be sorted in the order of file type > file name.
func (f *FileGateway) GetDeletedFiles(fileOwnerID string) ([]file.FileResponse, error) {
	query := fmt.Sprintf(
		`SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = ? AND %s IS NOT NULL
		ORDER BY %s, %s`,
		file.ColumnFileName, file.ColumnFileType,
		file.ColumnFileID, file.ColumnFileExtension,
		file.ColumnParentID, file.ColumnFileSize,
		file.ColumnModifiedOn, file.ColumnDeletedOn,
		file.ColumnUploadStatus, file.ColumnUniqueHash,
		file.TableName,
		file.ColumnFileOwnerID, file.ColumnDeletedOn,
		file.ColumnFileType, file.ColumnFileName,
	)

	args := []any{fileOwnerID}

	rows, err := f.database.Query(query, args...)
	if err != nil {
		return nil, logQueryError(f.deps.Log, err, query)
	}

	var files []file.FileResponse
	err = SelectRows(rows, &files)
	if err != nil {
		f.deps.Log.Criticalf("Failed to select rows from query: %v", err)
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
	now := utils.NowUTC()
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
		file.ColumnUploadStatus,
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

	fileIds := []string{}
	for _, f := range files {
		fileIds = append(fileIds, f.FileID)
	}
	f.deps.Log.Infof("Successfully added %d files", len(files))
	f.deps.Log.Debugf("Added file IDs: %s", strings.Join(fileIds, ","))
	logResultRows(f.deps.Log, res)

	return nil
}

// RenameFile renames a file to a new file name.
//
// The unique hash will be regenerated due to the name change. A duplication error can
// occur if the file is the 'file' type and it exists with the same parent ID.
// This will also update the modified date time to current time.
//
// It will return back the new File.
func (f *FileGateway) RenameFile(accountId, fileId, newFileName string) (*file.File, error) {
	q := fmt.Sprintf(`
		UPDATE %s
		SET %s = ?,
			%s = SHA2(CONCAT(%s, '%s', %s, %s), 256),
			%s = ?
		WHERE %s = ? AND %s = ?`,
		file.TableName,
		file.ColumnFileName,
		file.ColumnUniqueHash, file.ColumnFileOwnerID, newFileName, file.ColumnFileExtension, file.ColumnParentID,
		file.ColumnModifiedOn,
		file.ColumnFileOwnerID,
		file.ColumnFileID,
	)
	args := []any{newFileName, utils.NowUTC(), accountId, fileId}

	res, err := execQuery(f.database, q, args...)
	if IsDuplicateSqlError(err) {
		f.deps.Log.Warnf("Duplicate file rename for %s (-> %s)", fileId, newFileName)
		return nil, err
	}
	if err != nil {
		return nil, logQueryError(f.deps.Log, err, q)
	}

	f.deps.Log.Infof("Renamed file to %s (id=%s)", newFileName, fileId)
	logResultRows(f.deps.Log, res)

	fi, err := f.GetFile(accountId, fileId)
	if err != nil {
		f.deps.Log.Criticalf("Failed to retrieve file during file rename: %v", err)
		return nil, err
	}
	if fi == nil {
		f.deps.Log.Critical("File retrieval is empty")
		return nil, errors.New("failed to retrieve file")
	}

	return fi, nil
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

// DeleteFiles sets a slice of file ID's deletion date for a soft deletion.
// It will return the number of rows that were affected.
//
// It will set the file IDs for deletion by setting the date 15 days from the current
// date.
func (f *FileGateway) DeleteFiles(fileOwnerID string, fileIDs ...string) (int, error) {
	if len(fileIDs) == 0 {
		f.deps.Log.Critical("Failed to delete files, got file IDs length 0")
		return 0, ServerErr
	}

	const DAYS_UNTIL_DELETE = 15

	deleteTime := time.Now().AddDate(0, 0, DAYS_UNTIL_DELETE).UTC()
	query, args, err := sqlquery.Update(file.TableName, file.ColumnDeletedOn).Args(deleteTime).
		Where().Equal(file.ColumnFileOwnerID, fileOwnerID).
		And().In(file.ColumnFileID, utils.ConvertToAny(fileIDs)...).Build()
	if err != nil {
		return 0, logSqlBuildError(f.deps.Log, err, query, args)
	}

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute query: %v | Query: %s", err, query)
		return 0, SqlErr
	}

	logResultRows(f.deps.Log, res)

	// not sure what to do with the error here. i guess if a wrong DB is used, but
	// mysql supports this. so will just log and return nil?
	n, err := res.RowsAffected()
	if err != nil {
		f.deps.Log.Warnf("Failed to check affected rows: %v", err)
		return 0, nil
	}

	f.deps.Log.Debugf("Marked %d row(s) for deletion", n)

	return int(n), nil
}

// RestoreDeletedFiles sets file IDs' deletion column to NULL.
// It returns the number of rows that are affected and an error.
//
// If the parent folder is being deleted or does not exist with the given file ID,
// then the parent ID of the given file will be set to root.
func (f *FileGateway) RestoreDeletedFiles(fileOwnerID string, fileIDs ...string) (int, error) {
	// TODO: figure out a way to handle this update.
	// the main issue is the given file IDs can have different parent IDs
	query := fmt.Sprintf(`
		UPDATE %s f
		LEFT JOIN %s p
			ON p.%s = f.%s
			AND p.%s IS NULL
		SET
			f.%s = ?,
			f.%s = COALESCE(p.%s, '')
		WHERE f.%s = ? AND f.%s = ?`,
		file.TableName,
		file.TableName,
		file.ColumnFileID, file.ColumnParentID,
		file.ColumnDeletedOn,
		file.ColumnDeletedOn,
		file.ColumnParentID, file.ColumnFileID,
		file.ColumnFileOwnerID, file.ColumnFileID,
	)
	args := []any{nil, fileOwnerID}
	for _, s := range fileIDs {
		args = append(args, s)
	}

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		return 0, logQueryError(f.deps.Log, err, query)
	}

	n, err := res.RowsAffected()
	// mentioned this before, but this is supported with mysql
	// will just log and move on.
	if err != nil {
		f.deps.Log.Warnf("Failed to query rows for deleted files restoration: %v", err)
	}

	f.deps.Log.Debugf("Updated %d rows for restoration", n)

	return int(n), nil
}

// GetFilesByAccountIdAndParentId retrieves the files of a given folder ID. By default it will
// return the files sorted in the following order:
//   - file type (dir, file)
//   - file name
//
// If the given parent folder ID does not exist, it will return a 404 and an error.
//
// Deleted files are not included.
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

	// sorted by dir -> file name
	query := fmt.Sprintf(`
		SELECT f.*
		FROM %s f 
		JOIN %s 
			ON u.%s = f.%s 
		WHERE u.%s = ? AND f.%s = ? AND f.%s IS NULL
		ORDER BY f.%s, f.%s
		`,
		file.TableName,
		fmt.Sprintf("%s u", user.TableName),
		user.ColumnAccountID,
		file.ColumnFileOwnerID,
		user.ColumnAccountID,
		file.ColumnParentID,
		file.ColumnDeletedOn,
		file.ColumnFileType,
		file.ColumnFileName,
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

// GetBreadcrumbs retrieves all the parent folders from a given folder ID. It will
// perform a recursive CTE to retrieve all file types related to the given folder ID.
// The slice will be returned in the following order:
//   - root folder
//   - [if existing] parents between the two
//   - given folder ID
//
// Assuming the given folder ID exists, there will be a minimum one entry which is the
// query of the original folder ID. If parents exists for the folder ID, the parent of the folder ID
// and its parents are recursively retrieved starting from the parent. The first and final entries
// of the slice will always be the original folder ID and the root folder ID.
//
// The file type must be a type 'dir'.
func (f *FileGateway) GetBreadcrumbs(accountId, folderId string) ([]BreadcrumbFile, error) {
	query := fmt.Sprintf(`
		WITH RECURSIVE parent_files AS (
			SELECT %s, %s, %s
			FROM %s
			WHERE %s = ? AND %s = ? AND %s = ?

			UNION ALL

			SELECT fc.%s, fc.%s, fc.%s
			FROM %s fc
			JOIN parent_files fp ON fc.%s = fp.%s
		)
		SELECT %s, %s, %s FROM parent_files;`,
		file.ColumnFileID, file.ColumnParentID, file.ColumnFileName,
		file.TableName,
		file.ColumnFileID, file.ColumnFileOwnerID, file.ColumnFileType,
		file.ColumnFileID, file.ColumnParentID, file.ColumnFileName,
		file.TableName,
		file.ColumnFileID, file.ColumnParentID,
		file.ColumnFileID, file.ColumnParentID, file.ColumnFileName,
	)

	args := []any{folderId, accountId, file.FileTypeDir}

	rows, err := f.database.Query(query, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to execute query for recursive parent folders: %v | Query: %s", err, query)
		return nil, err
	}

	var files []BreadcrumbFile
	err = SelectRows(rows, &files)
	if err != nil {
		f.deps.Log.Criticalf("Failed to parse FileFolderInfo rows: %v", err)
		return nil, err
	}

	f.deps.Log.Debugf("Breadcrumb rows found: %d", len(files))

	// reverse for breadcrumbs
	slices.Reverse(files)

	return files, nil
}

// UpdateUploadStatus updates the upload status to a given status.
//
// This does not update the modified on time.
func (f *FileGateway) UpdateUploadStatus(accountId, fileId string, status file.UploadStatus) error {
	q, args, err := sqlquery.Update(file.TableName, file.ColumnUploadStatus).Args(status).
		Where().Equal(file.ColumnFileOwnerID, accountId).And().Equal(file.ColumnFileID, fileId).Build()
	if err != nil {
		return logSqlBuildError(f.deps.Log, err, q, args)
	}

	res, err := execQuery(f.database, q, args...)
	if err != nil {
		return logQueryError(f.deps.Log, err, q)
	}

	f.deps.Log.Infof("Updated file (%s) upload status to '%s'", fileId, status)
	logResultRows(f.deps.Log, res)

	return nil
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
			&f.UploadStatus,
			&f.UniqueHash,
		)

		if scanErr != nil {
			return nil, scanErr
		}

		files = append(files, f)
	}

	return files, nil
}
