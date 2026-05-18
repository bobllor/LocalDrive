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

// GetAllFiles returns a File slice of all File rows belonging to the file owner.
//
// If an error occurs then it will return an error, and abort
// the scanning process if it is occurring.
func (f *FileGateway) GetAllFiles(fileOwnerID string) ([]file.FileResponse, error) {
	query, args, err := sqlquery.Select(
		file.TableName,
		file.ColumnFileName, file.ColumnFileType,
		file.ColumnFileID, file.ColumnParentID,
		file.ColumnFileSize, file.ColumnModifiedOn,
		file.ColumnDeletedOn,
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

// GetFile retrieves a single file based on the given file ID.
//
// If the file does not exist, it will return nil. This must be handled.
func (f *FileGateway) GetFile(fileOwnerId string, fileId string) (*file.FileResponse, error) {
	q, args, err := sqlquery.Select(
		file.TableName,
		file.ColumnFileName, file.ColumnFileType,
		file.ColumnFileID, file.ColumnParentID,
		file.ColumnFileSize, file.ColumnModifiedOn,
		file.ColumnDeletedOn,
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

	var fr []file.FileResponse
	err = SelectRows(rows, &fr)
	if err != nil {
		f.deps.Log.Criticalf("Failed to retrieve data from query: %v | Query: %s", err, q)
		return nil, SqlErr
	}
	if len(fr) == 0 {
		return nil, nil
	}

	return &fr[0], nil
}

// GetFiles returns File rows based on the clause and conditions.
func (f *FileGateway) GetFiles(fileOwnerID string, conditions []WhereCondition) ([]file.File, error) {
	cb := NewClauseBuilder()

	baseQuery := fmt.Sprintf("SELECT * FROM %s", file.TableName)

	cb.Equal(file.ColumnFileOwnerID, fileOwnerID)

	err := cb.RegisterConditions(conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to register conditions: %v", err)
	}

	q, args, err := cb.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %v", err)
	}

	query := baseQuery + " " + q

	rows, err := f.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v | query: %s", err, query)
	}

	files, err := f.getFiles(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %v", err)
	}

	return files, nil
}

// UpdateFileByID updates a single file row by its file ID.
//
// cd is a ClauseData type used to target the columns and values to replace for the row.
func (f *FileGateway) UpdateFileByID(fileOwnerID string, fileID string, cd ClauseData) error {
	conditions := []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{fileID},
			ComparisonOperator: Equal,
			LogicalOperator:    OperatorAnd,
		},
	}

	err := f.UpdateFiles(fileOwnerID, cd, conditions)
	if err != nil {
		return err
	}

	return nil
}

// UpdateFiles updates the Files table based ClauseData and conditions.
//
// Errors will be returned if one occurs.
// Certain columns are forbidden from being changed, and will return an error
// if these are found.
func (f *FileGateway) UpdateFiles(fileOwnerID string, cd ClauseData, conditions []WhereCondition) error {
	cb := NewClauseBuilder()

	setQ, sargs, err := cd.BuildSetQuery()
	if err != nil {
		return fmt.Errorf("failed to validate ClauseData: %v", err)
	}

	baseQuery := fmt.Sprintf("UPDATE %s %s", file.TableName, setQ)

	cb.Equal(file.ColumnFileOwnerID, fileOwnerID)

	err = cb.RegisterConditions(conditions)
	if err != nil {
		return fmt.Errorf("failed to register conditions: %v", err)
	}

	whereQ, args, err := cb.Build()
	if err != nil {
		return fmt.Errorf("failed to build WHERE clause: %v", err)
	}

	query := baseQuery + " " + whereQ

	execArgs := MakeArgs(sargs, args)

	res, err := execQuery(f.database, query, execArgs...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v (args: %v) | query: %s", err, execArgs, query)
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// AddFile adds slice of File structs to the File database.
// If an error occurs it will return an error.
//
// This does not write the files to the disk.
func (f *FileGateway) AddFile(files []file.File) error {
	if len(files) == 0 {
		return fmt.Errorf("no arguments given for AddFile")
	}

	flatFiles := file.FlattenFile(files...)

	query := fmt.Sprintf("INSERT INTO %s VALUES", file.TableName)
	paramStr := BuildPlaceholder(f.fileFieldCount, len(files))

	query = query + " " + paramStr

	res, err := execQuery(f.database, query, flatFiles...)
	if err != nil {
		return fmt.Errorf("failed to insert into %s: %v | query: %s", file.TableName, err, query)
	}

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
		f.deps.Log.Criticalf("Failed to build query to update: %v | Query: %s | Args: %d", err, q, len(args))
		return errors.New("an error occurred on the server")
	}

	res, err := execQuery(f.database, q, args...)
	if err != nil {
		f.deps.Log.Criticalf("Failed to update file name: %v | query: %s", err, q)
		return fmt.Errorf("failed to update file name: %v", err)
	}
	logResultRows(f.deps.Log, res)

	return nil
}

// UpdateModifiedFiles updates the modified date column to the current time.
func (f *FileGateway) UpdateModifiedFiles(fileOwnerID string, fileIDs []string) error {
	cb := NewClauseBuilder()

	convIds := utils.ConvertToAny(fileIDs)

	cb.Equal(file.ColumnFileOwnerID, fileOwnerID).And().In(file.ColumnFileID, convIds...)

	qCon, args, err := cb.Build()
	if err != nil {
		return fmt.Errorf("failed to build query: %v", err)
	}

	query := fmt.Sprintf("UPDATE %s SET %s = ?", file.TableName, file.ColumnModifiedOn) + " " + qCon

	finalArgs := []any{time.Now().Format(time.DateTime)}
	finalArgs = append(finalArgs, args...)

	res, err := execQuery(f.database, query, finalArgs...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v | query: %s", err, query)
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// DeleteFile sets a slice of file IDs to be marked for deletion.
// This does not delete the files immediately.
func (f *FileGateway) DeleteFiles(fileOwnerID string, fileIDs []string) error {
	cb := NewClauseBuilder()

	if len(fileIDs) == 0 {
		return fmt.Errorf("failed to delete files, got empty file IDs")
	}

	convFileIDs := utils.ConvertToAny(fileIDs)

	cb.Equal(file.ColumnFileOwnerID, fileOwnerID).And().In(file.ColumnFileID, convFileIDs...)

	qCondition, args, err := cb.Build()
	if err != nil {
		return fmt.Errorf("failed to build condition query: %v", err)
	}

	baseQuery := fmt.Sprintf(
		"UPDATE %s SET %s = ?",
		file.TableName,
		file.ColumnDeletedOn,
	)
	query := baseQuery + " " + qCondition

	finalArgs := []any{time.Now().Format(time.DateTime)}
	finalArgs = append(finalArgs, args...)

	res, err := execQuery(f.database, query, finalArgs...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v | query: %s", err, query)
	}

	logResultRows(f.deps.Log, res)

	return nil
}

// RestoreFiles sets a file IDs that are unmark files that were marked for deletion.
func (f *FileGateway) RestoreFiles(fileOwnerID string, fileIDs []string) error {
	cb := NewClauseBuilder()

	cb.Equal(file.ColumnFileOwnerID, fileOwnerID)

	convIDs := utils.ConvertToAny(fileIDs)

	cb.And().In(file.ColumnFileID, convIDs...)

	cond, args, err := cb.Build()
	if err != nil {
		return fmt.Errorf("failed to build conditions: %v", err)
	}

	baseQuery := fmt.Sprintf("UPDATE %s SET %s = NULL", file.TableName, file.ColumnDeletedOn)
	query := baseQuery + " " + cond

	res, err := execQuery(f.database, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v | query: %s", err, query)
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

	args := []any{accountId}
	parentCondition := "= ?"
	if parentFolderID == "" {
		parentCondition = "IS NULL"
	} else {
		args = append(args, parentFolderID)
	}

	query := fmt.Sprintf(`
		SELECT f.*
		FROM %s f 
		JOIN %s 
			ON u.%s = f.%s 
		WHERE u.%s = ? AND f.%s %s
		`,
		file.TableName,
		fmt.Sprintf("%s u", user.TableName),
		user.ColumnAccountID,
		file.ColumnFileOwnerID,
		user.ColumnAccountID,
		file.ColumnParentID,
		parentCondition,
	)

	rows, err := f.database.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %v | query: %s", err, query)
	}

	files, err := f.getFiles(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to parse File query: %v", err)
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
		WHERE u.%s = ? AND f.%s = ? AND f.%s = ?
		`,
		file.TableName,
		user.TableName,
		user.ColumnAccountID,
		file.ColumnFileOwnerID,
		user.ColumnAccountID,
		file.ColumnFileID,
		file.ColumnFileType,
	)

	rows, err := f.database.Query(query, accountID, fileID, file.FileTypeDir)
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
			&f.ParentID,
			&f.Path,
			&f.Size,
			&f.ModifiedOn,
			&f.DeletedOn,
		)

		if scanErr != nil {
			return nil, scanErr
		}

		files = append(files, f)
	}

	return files, nil
}
