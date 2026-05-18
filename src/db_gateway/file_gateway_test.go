package dbgateway

import (
	"fmt"
	"testing"
	"time"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/utils"
)

// IMPORTANT: These tests require the test database to exist.
// See the Docker setup documentation.
//
// By default when the test Docker setup is ran, one row is added to both
// the UserAccount and File table by default.
// As rows are added into the table, it will grow over the course of the test cases.
// Majority of the tests with modifications only affects the default rows.
// Be aware of it!

func TestGetAllFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	files, err := fDb.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	assert.NotEqual(t, len(files), 0)
}

func TestGetFile(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	conditions := []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{tests.DbRowInfo.FileID},
			LogicalOperator:    OperatorAnd,
			ComparisonOperator: Equal,
		},
	}

	qFiles, err := fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	assert.Equal(t, len(qFiles), 1)
}

func TestGetSingleFile(t *testing.T) {
	fg, err := getTestFileGateway()
	assert.Nil(t, err)

	t.Run("Normal use", func(t *testing.T) {
		f, err := fg.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
		assert.Nil(t, err)

		assert.NotNil(t, f)
		assert.Equal(t, f.Name, tests.DbRowInfo.FileName)
	})

	t.Run("Empty file", func(t *testing.T) {
		f, err := fg.GetFile(tests.DbRowInfo.AccountID, "1234five.txt")
		assert.Nil(t, err)

		assert.Nil(t, f)
	})

	t.Run("Bad account ID", func(t *testing.T) {
		f, err := fg.GetFile("nonexistentid", tests.DbRowInfo.FileID)
		assert.Nil(t, err)

		assert.Nil(t, f)
	})
}

func TestAddFile(t *testing.T) {
	dir := t.TempDir()

	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	_, err = tests.CreateFiles(dir)
	assert.Nil(t, err)

	files, err := file.Read(dir)
	assert.Nil(t, err)

	fileIDs := []string{}
	// File.OwnerID is nil, this is changed to the existing account ID by default.
	for i := range files {
		files[i].OwnerID = tests.DbRowInfo.AccountID

		fileIDs = append(fileIDs, files[i].FileID)
	}

	err = fDb.AddFile(files)
	assert.Nil(t, err)

	qFiles, err := fDb.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	// only 1 row exists by default, afterwards it adds however many from files
	assert.NotEqual(t, len(qFiles), 1)
	assert.NotEqual(t, len(qFiles), 0)

	_, err = DropRows(fDb.database, file.TableName, file.ColumnFileID, utils.ConvertToAny(fileIDs)...)
	assert.Nil(t, err)
}

func TestUpdateFileByID(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	newValue := "this.is.a.text.file.txt"

	cd := ClauseData{
		Columns: []string{file.ColumnFileName},
		Args:    []any{newValue},
	}

	err = fDb.UpdateFileByID(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, cd)
	assert.Nil(t, err)

	files, err := fDb.GetFiles(tests.DbRowInfo.AccountID, getConditionByID(tests.DbRowInfo.FileID))
	assert.Nil(t, err)

	assert.Equal(t, files[0].Name, newValue)

	err = resetDefaultFileRow(fDb, file.ColumnFileName, tests.DbRowInfo.FileName)
	assert.Nil(t, err)
}

func TestDeleteFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	err = fDb.DeleteFiles(tests.DbRowInfo.AccountID, []string{tests.DbRowInfo.FileID})
	assert.Nil(t, err)

	conditions := []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{tests.DbRowInfo.FileID},
			LogicalOperator:    OperatorAnd,
			ComparisonOperator: Equal,
		},
	}

	qFiles, err := fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	assert.NotNil(t, qFiles[0].DeletedOn)
	assert.Equal(t, qFiles[0].FileID, tests.DbRowInfo.FileID)

	now := time.Now()
	qDate := qFiles[0].DeletedOn

	assert.Equal(t, qDate.Year(), now.Year())
	assert.Equal(t, qDate.Month(), now.Month())
	assert.Equal(t, qDate.Day(), now.Day())

	err = resetDefaultFileRow(fDb, file.ColumnDeletedOn, nil)
	assert.Nil(t, err)
}

func TestRestoreFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	conditions := []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{tests.DbRowInfo.FileID},
			LogicalOperator:    OperatorAnd,
			ComparisonOperator: Equal,
		},
	}

	err = fDb.DeleteFiles(tests.DbRowInfo.AccountID, []string{tests.DbRowInfo.FileID})
	assert.Nil(t, err)

	qFiles, err := fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	if qFiles[0].DeletedOn == nil {
		t.Fatal("failed to set file to deleted with DeletedOn")
	}

	err = fDb.RestoreFiles(tests.DbRowInfo.AccountID, []string{tests.DbRowInfo.FileID})
	assert.Nil(t, err)

	qFiles, err = fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	assert.Nil(t, qFiles[0].DeletedOn)
}

func TestUpdateModifiedFile(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	conditions := []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{tests.DbRowInfo.FileID},
			LogicalOperator:    OperatorAnd,
			ComparisonOperator: Equal,
		},
	}

	baseFiles, err := fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	baseDate := baseFiles[0].ModifiedOn

	err = fDb.UpdateModifiedFiles(tests.DbRowInfo.AccountID, []string{tests.DbRowInfo.FileID})
	assert.Nil(t, err)

	newFiles, err := fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	newDate := newFiles[0].ModifiedOn

	fmt.Println(newDate, baseDate)
	assert.Equal(t, baseDate.Compare(newDate), -1)

	err = resetDefaultFileRow(fDb, file.ColumnModifiedOn, baseDate)
	assert.Nil(t, err)
}

func TestAddDuplicateFileError(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	f := file.File{
		OwnerID: tests.DbRowInfo.AccountID,
		FileID:  tests.DbRowInfo.FileID,
	}

	err = fDb.AddFile([]file.File{f})
	assert.NotNil(t, err)
}

func TestAddMissingOwnerIDFileError(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	f := file.File{
		FileID: "fdsa",
	}

	err = fDb.AddFile([]file.File{f})
	assert.NotNil(t, err)
}

func TestUpdateFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	newName := "this.isa.filename.txt"

	cd := ClauseData{
		Columns: []string{file.ColumnFileName},
		Args:    []any{newName},
	}

	conditions := []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{tests.DbRowInfo.FileID},
			ComparisonOperator: Equal,
			LogicalOperator:    OperatorAnd,
		},
	}

	files, err := fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	baseName := files[0].Name

	err = fDb.UpdateFiles(tests.DbRowInfo.AccountID, cd, conditions)
	assert.Nil(t, err)

	files, err = fDb.GetFiles(tests.DbRowInfo.AccountID, conditions)
	assert.Nil(t, err)

	assert.Equal(t, files[0].Name, newName)
	assert.NotEqual(t, files[0].Name, baseName)

	err = resetDefaultFileRow(fDb, file.ColumnFileName, tests.DbRowInfo.FileName)
	assert.Nil(t, err)
}

func TestGetFilesByAccountIDAndParentFolder(t *testing.T) {
	fg, err := getTestFileGateway()
	assert.Nil(t, err)

	t.Run("Root folder", func(t *testing.T) {
		files, err := fg.GetFilesByAccountIdAndParentId(tests.DbRowInfo.AccountID, "")
		assert.Nil(t, err)

		assert.Equal(t, len(files), 2)
	})

	t.Run("Child folder", func(t *testing.T) {
		// not located in tests.DbRowInfo, obtained from the test SQL script
		parent := "randomfolderidhere"
		baseName := "test2.txt"
		files, err := fg.GetFilesByAccountIdAndParentId(tests.DbRowInfo.AccountID, parent)
		assert.Nil(t, err)

		assert.Equal(t, len(files), 1)
		assert.Equal(t, files[0].Name, baseName)
	})

	t.Run("Invalid folder", func(t *testing.T) {
		parent := "doesnotexist"

		_, err := fg.GetFilesByAccountIdAndParentId(tests.DbRowInfo.AccountID, parent)
		assert.NotNil(t, err)
		assert.Equal(t, err, FileDoesNotExistErr)
	})
}

func TestValidateFileExists(t *testing.T) {
	gw, err := getTestFileGateway()
	assert.Nil(t, err)

	t.Run("File exists", func(t *testing.T) {
		stat, err := gw.validateFileExists(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
		assert.Nil(t, err)

		assert.True(t, stat)
	})

	t.Run("File not exists", func(t *testing.T) {
		stat, err := gw.validateFileExists(tests.DbRowInfo.AccountID, "12345")
		assert.Nil(t, err)

		assert.False(t, stat)
	})
}

func TestRenameFileName(t *testing.T) {
	fg, err := getTestFileGateway()
	assert.Nil(t, err)
	newFileName := "testfile"

	t.Cleanup(func() {
		UpdateRow(
			fg.database,
			file.TableName,
			file.ColumnFileID,
			tests.DbRowInfo.FileID,
			ClauseData{
				Columns: []string{file.ColumnFileName},
				Args:    []any{tests.DbRowInfo.FileName},
			},
		)
	})

	err = fg.RenameFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, newFileName)
	assert.Nil(t, err)
}

// getFileDb gets the [FileGateway] for the test database.
// If an error occurs, it will return an error.
//
// This function does not start the test database instance.
func getTestFileGateway() (*FileGateway, error) {
	dbConfig := newTestDBConfig()
	db, err := NewDatabase(dbConfig)
	if err != nil {
		return nil, err
	}

	deps := utils.NewTestDeps()

	fDb := NewFileGateway(db, deps)

	return fDb, nil
}

func getConditionByID(fileID string) []WhereCondition {
	return []WhereCondition{
		{
			Column:             file.ColumnFileID,
			Args:               []any{fileID},
			ComparisonOperator: Equal,
			LogicalOperator:    OperatorAnd,
		},
	}
}

// getClauseData retrieves a default ClauseData that targets
// the a given column and given arguments.
func getClauseData(column string, args ...any) ClauseData {
	return ClauseData{
		Columns: []string{column},
		Args:    args,
	}
}

// resetDefaultFileName resets the default entry's file name to its default value.
// The error must be handled.
func resetDefaultFileRow(fDb *FileGateway, column string, args ...any) error {
	cd := getClauseData(column, args...)

	err := fDb.UpdateFileByID(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, cd)
	if err != nil {
		return err
	}
	return nil
}
