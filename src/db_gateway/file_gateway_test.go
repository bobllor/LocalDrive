package dbgateway

import (
	"testing"
	"time"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/user"
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

	err = fDb.AddFile(files...)
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

func TestAddFileDuplicate(t *testing.T) {
	gw, db := NewTestGatewayDB(t)

	t.Run("Duplicate file error", func(t *testing.T) {
		fi := file.NewFile(
			tests.DbRowInfo.AccountID,
			tests.DbRowInfo.FileName,
			file.FileTypeFile,
			"txt",
			0,
			"",
			file.UploadPending,
		)

		// not needed for this test run but using just in case
		t.Cleanup(func() {
			DropRows(db, file.TableName, file.ColumnFileID, fi.FileID)
		})

		err := gw.File.AddFile(fi)
		assert.NotNil(t, err)
		assert.True(t, IsDuplicateSqlError(err))
	})

	t.Run("Duplicate name pass diff parent ID", func(t *testing.T) {
		fi := file.NewFile(
			tests.DbRowInfo.AccountID,
			tests.DbRowInfo.FileName,
			file.FileTypeFile,
			"txt",
			0,
			"parentid1",
			file.UploadPending,
		)

		t.Cleanup(func() {
			DropRows(db, file.TableName, file.ColumnFileID, fi.FileID)
		})

		err := gw.File.AddFile(fi)
		assert.Nil(t, err)

		_, err = gw.File.GetFile(tests.DbRowInfo.AccountID, fi.FileID)
		assert.Nil(t, err)
	})

	t.Run("Duplicate name pass different account ID", func(t *testing.T) {
		username := "iamauser"
		usr, err := gw.User.AddUser(username, "1234!password")
		assert.Nil(t, err)

		t.Cleanup(func() {
			// cascade deletion
			DropRows(db, user.TableName, user.ColumnAccountID, usr.AccountID)
		})

		fi := file.NewFile(
			usr.AccountID,
			tests.DbRowInfo.FileName,
			file.FileTypeFile,
			"txt",
			0,
			"",
			file.UploadPending,
		)

		err = gw.File.AddFile(fi)
		assert.Nil(t, err)

		bfi, err := gw.File.GetFile(usr.AccountID, fi.FileID)
		assert.Nil(t, err)
		assert.NotNil(t, bfi)

		assert.Equal(t, bfi.Name, fi.Name)
		assert.Equal(t, bfi.Extension, fi.Extension)
	})

	t.Run("Add folders no duplicate error", func(t *testing.T) {
		folderName := "a folder here"
		folder1 := file.NewFile(
			tests.DbRowInfo.AccountID,
			folderName,
			file.FileTypeDir,
			"",
			0,
			"",
			file.UploadCompleted,
		)

		folder2 := file.NewFile(
			tests.DbRowInfo.AccountID,
			folderName,
			file.FileTypeDir,
			"",
			0,
			"",
			file.UploadCompleted,
		)

		t.Cleanup(func() {
			DropRows(db, file.TableName, file.ColumnFileID, folder1.FileID)
			DropRows(db, file.TableName, file.ColumnFileID, folder2.FileID)
		})

		err := gw.File.AddFile(folder1, folder2)
		assert.Nil(t, err)

		bf1, err := gw.File.GetFile(tests.DbRowInfo.AccountID, folder1.FileID)
		assert.Nil(t, err)
		bf2, err := gw.File.GetFile(tests.DbRowInfo.AccountID, folder2.FileID)
		assert.Nil(t, err)

		assert.Equal(t, bf1.Name, folderName)
		assert.Equal(t, bf2.Name, folderName)
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

func TestAddDuplicateFile(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	cases := []struct {
		f     file.File
		isErr bool
		name  string
	}{
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileName,
				file.FileTypeFile, "txt", 0, "", file.UploadCompleted),
			isErr: true,
			name:  "Duplicate error existing file",
		},
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileName,
				file.FileTypeFile, "txt", 0, "", file.UploadFailed),
			name: "Duplicate success failed upload",
		},
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileName,
				file.FileTypeFile, "txt", 0, "", file.UploadPending),
			isErr: true,
			name:  "Duplicate error pending existing file",
		},
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileName,
				file.FileTypeDir, "", 0, "", file.UploadCompleted),
			name: "Duplicate folder name success",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err = fDb.AddFile(c.f)

			t.Cleanup(func() {
				DropRows(fDb.database, file.TableName, file.ColumnFileID, c.f.FileID)
			})

			if c.isErr {
				assert.NotNil(t, err)
				assert.True(t, IsDuplicateSqlError(err))
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

func TestAddMissingOwnerIDFileError(t *testing.T) {
	fDb, err := getTestFileGateway()
	assert.Nil(t, err)

	f := file.File{
		FileID:     "fdsa",
		ModifiedOn: time.Now().UTC(),
	}

	err = fDb.AddFile(f)
	assert.NotNil(t, err)
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

func TestRenameDuplicateFiles(t *testing.T) {
	gw, db := NewTestGatewayDB(t)

	cases := []struct {
		f           file.File
		isErr       bool
		newFileName string
		name        string
	}{
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, "afilename", file.FileTypeFile, "txt",
				0, "", file.UploadCompleted),
			newFileName: tests.DbRowInfo.FileName,
			isErr:       true,
			name:        "Rename file duplicate error",
		},
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, "afilename", file.FileTypeFile, "txt",
				0, "", file.UploadCompleted),
			newFileName: tests.DbRowInfo.FileName,
			isErr:       true,
			name:        "Rename file pending duplicate error",
		},
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, "afilename", file.FileTypeFile, "txt",
				0, "12345", file.UploadCompleted),
			newFileName: tests.DbRowInfo.FileName,
			name:        "Rename file success different parent ID",
		},
		{
			f: file.NewFile(tests.DbRowInfo.AccountID, "afilename", file.FileTypeFile, "txt",
				0, "12345", file.UploadCompleted),
			newFileName: "different file name",
			name:        "Rename file success",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Cleanup(func() {
				DropRows(db, file.TableName, file.ColumnFileID, c.f.FileID)
			})

			err := gw.File.AddFile(c.f)
			assert.Nil(t, err)

			err = gw.File.RenameFile(tests.DbRowInfo.AccountID, c.f.FileID, c.newFileName)
			if c.isErr {
				assert.NotNil(t, err)
				assert.True(t, IsDuplicateSqlError(err))
			} else {
				assert.Nil(t, err)
				fi, err := gw.File.GetFile(c.f.OwnerID, c.f.FileID)
				assert.Nil(t, err)
				assert.NotNil(t, fi)
				assert.Equal(t, fi.Name, c.newFileName)
			}
		})
	}
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
		baseName := "test2"
		files, err := fg.GetFilesByAccountIdAndParentId(tests.DbRowInfo.AccountID, parent)
		assert.Nil(t, err)

		assert.Equal(t, len(files), 1)
		assert.Equal(t, files[0].Name, baseName)
		assert.Equal(t, files[0].Extension, "txt")
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
			[]string{file.ColumnFileName, file.ColumnUniqueHash},
			tests.DbRowInfo.FileName,
			tests.DbRowInfo.UniqueHash,
		)
	})

	// confirming the hash is changed
	err = fg.RenameFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, newFileName)
	assert.Nil(t, err)
	fi, err := fg.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)

	assert.NotEqual(t, fi.UniqueHash, tests.DbRowInfo.UniqueHash)

	// confirming the hash is back to its default value
	err = fg.RenameFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID, tests.DbRowInfo.FileName)
	assert.Nil(t, err)
	fi, err = fg.GetFile(tests.DbRowInfo.AccountID, tests.DbRowInfo.FileID)
	assert.Nil(t, err)
	assert.NotNil(t, fi)

	assert.Equal(t, fi.UniqueHash, tests.DbRowInfo.UniqueHash)
}

func TestUpdateUploadStatus(t *testing.T) {
	gw, db := NewTestGatewayDB(t)

	f1 := file.NewFile(tests.DbRowInfo.AccountID, "filename1234", file.FileTypeFile,
		"pdf", 0, "", file.UploadPending)

	t.Cleanup(func() {
		DropRows(db, file.TableName, file.ColumnFileID, f1.FileID)
	})

	err := gw.File.AddFile(f1)
	assert.Nil(t, err)

	err = gw.File.UpdateUploadStatus(tests.DbRowInfo.AccountID, f1.FileID, file.UploadFailed)
	assert.Nil(t, err)

	bf, err := gw.File.GetFile(tests.DbRowInfo.AccountID, f1.FileID)
	assert.Nil(t, err)
	assert.NotNil(t, bf)

	assert.Equal(t, string(bf.UploadStatus), string(file.UploadFailed))
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
