package dbgateway

import (
	"errors"
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

	qFiles, err := fDb.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	assert.True(t, len(qFiles) > 0)
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

	defer t.Cleanup(func() {
		DropRows(fDb.database, file.TableName, file.ColumnFileID, utils.ConvertToAny(fileIDs)...)
	})
}

func TestDeleteFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	t.Cleanup(func() {
		setDefaultFileColumn(fDb, file.ColumnDeletedOn, nil)
	})

	err = fDb.DeleteFiles(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	qFile, err := fDb.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	assert.NotNil(t, qFile.DeletedOn)
	assert.Equal(t, qFile.FileID, tests.DbRowInfo.FileID)

	now := time.Now()
	qDate := qFile.DeletedOn

	expectedTime := now.AddDate(0, 0, 15).UTC()

	assert.Equal(t, qDate.Year(), expectedTime.Year())
	assert.Equal(t, qDate.Month(), expectedTime.Month())
	assert.Equal(t, qDate.Day(), expectedTime.Day())
}

func TestRestoreFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	err = fDb.DeleteFiles(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	qFile, err := fDb.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	assert.NotNil(t, qFile.DeletedOn)

	err = fDb.RestoreFiles(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	qFile, err = fDb.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	assert.Nil(t, qFile.DeletedOn)
}

func TestUpdateModifiedFile(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)
	baseFiles, err := fDb.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	baseDate := baseFiles[0].ModifiedOn

	t.Cleanup(func() {
		setDefaultFileColumn(fDb, file.ColumnModifiedOn, baseDate)
	})

	err = fDb.UpdateModifiedFiles(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	newFile, err := fDb.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	newDate := newFile.ModifiedOn

	assert.Equal(t, baseDate.Compare(newDate), -1)
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
		FileID:     "fdsa",
		ModifiedOn: time.Now().UTC(),
	}

	err = fDb.AddFile([]file.File{f})
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, SqlErr))
}

func TestUpdateFiles(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	t.Cleanup(func() {
		setDefaultFileColumn(fDb, file.ColumnFileName, tests.DbRowInfo.FileName)
	})

	newName := "this.isa.filename.txt"

	files, err := fDb.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	baseFile := files[0]

	baseTime := baseFile.ModifiedOn.UTC()
	baseName := baseFile.Name

	err = fDb.UpdateFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, file.ColumnFileName, newName)
	assert.Nil(t, err)

	fileRes, err := fDb.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	assert.Equal(t, fileRes.Name, newName)
	assert.NotEqual(t, fileRes.Name, baseName)

	assert.Equal(t, baseTime.UTC().Compare(fileRes.ModifiedOn), -1)
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
			[]string{file.ColumnFileName},
			tests.DbRowInfo.FileName,
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

// setDefaultFileColumn sets a column and arg to the default file row.
// It automatically targets the default test account ID and file ID.
func setDefaultFileColumn(fg *FileGateway, column string, arg any) error {
	err := fg.UpdateFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, column, arg)
	if err != nil {
		return err
	}

	return nil
}
