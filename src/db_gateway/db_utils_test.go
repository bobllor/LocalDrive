package dbgateway

import (
	"fmt"
	"testing"
	"time"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/sqlquery"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/user"
)

func TestSelectRow(t *testing.T) {
	dbConfig := newTestDBConfig()

	db, err := NewDatabase(dbConfig)
	assert.Nil(t, err)

	query := fmt.Sprintf(
		"SELECT %s, %s, %s FROM %s WHERE %s = ?",
		file.ColumnFileName,
		file.ColumnFileID,
		file.ColumnFileSize,
		file.TableName,
		file.ColumnFileOwnerID,
	)

	type FileFilterTest struct {
		FileName string
		FileID   string
		FileSize int
	}

	ffTest := FileFilterTest{}

	rows, err := db.Query(query, tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	err = SelectRow(rows, &ffTest)
	assert.Nil(t, err)

	assert.NotEqual(t, ffTest.FileName, "")
	assert.NotEqual(t, ffTest.FileID, "")
}

func TestSelectRowsSlice(t *testing.T) {
	dbConfig := newTestDBConfig()

	db, err := NewDatabase(dbConfig)
	assert.Nil(t, err)

	query := fmt.Sprintf(
		"SELECT %s, %s, %s, %s, %s FROM %s WHERE %s = ?",
		file.ColumnFileName,
		file.ColumnFileID,
		file.ColumnFileSize,
		file.ColumnModifiedOn,
		file.ColumnDeletedOn,
		file.TableName,
		file.ColumnFileOwnerID,
	)

	type FileFilterTest struct {
		FileName   string
		FileID     string
		FileSize   int
		ModifiedOn time.Time
		DeletedOn  *time.Time
	}

	ffTest := []FileFilterTest{}

	rows, err := db.Query(query, tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	err = SelectRows(rows, &ffTest)
	assert.Nil(t, err)
}

func TestMultipleSelectRows(t *testing.T) {
	fdb, err := getTestFileGateway()
	assert.Nil(t, err)

	root := t.TempDir()

	_, err = tests.CreateFiles(root)
	assert.Nil(t, err)

	fileIDs := []any{}

	files, err := file.Read(root)
	assert.Nil(t, err)

	ug := newTestUserGateway(t)
	usr, err := ug.AddUser("username123", "password12345")
	assert.Nil(t, err)

	for i, file := range files {
		files[i].OwnerID = usr.AccountID

		fileIDs = append(fileIDs, file.FileID)
	}

	defer t.Cleanup(func() {
		DropRows(fdb.database, user.TableName, user.ColumnAccountID, usr.AccountID)
	})

	err = fdb.AddFile(files)
	assert.Nil(t, err)

	type MultipleFileColumns struct {
		FileName string
		FileID   string
	}

	query, args, err := sqlquery.Select(file.TableName, file.ColumnFileName, file.ColumnFileID).
		Where().In(file.ColumnFileID, fileIDs...).Build()
	assert.Nil(t, err)

	rows, err := fdb.database.Query(query, args...)
	assert.Nil(t, err)

	data := []MultipleFileColumns{}
	err = SelectRows(rows, &data)
	assert.Nil(t, err)

	assert.Equal(t, len(data), len(files))
}

func TestFailSelectRowsNilRows(t *testing.T) {
	v := []file.File{}

	err := SelectRows(nil, &v)
	assert.NotNil(t, err)
}

func TestFailSelectRowsNonPointer(t *testing.T) {
	v := []user.UserAccount{}

	udb := newTestUserGateway(t)

	query := fmt.Sprintf("SELECT * FROM %s", user.TableName)

	rows, err := udb.database.Query(query)
	assert.Nil(t, err)

	err = SelectRows(rows, v)
	assert.NotNil(t, err)
}

func TestFailSelectRowsInvalidSize(t *testing.T) {
	v := []user.UserAccount{}

	udb := newTestUserGateway(t)

	query := fmt.Sprintf("SELECT %s FROM %s", user.ColumnAccountID, user.TableName)

	rows, err := udb.database.Query(query)
	assert.Nil(t, err)

	err = SelectRows(rows, &v)
	assert.NotNil(t, err)
}

func TestMakeArgs(t *testing.T) {
	s1 := []string{"1", "2", "3"}
	s2 := []int{1, 2, 3}
	s3 := []bool{true, true, false}

	t.Run("Slice Arguments Only", func(t *testing.T) {
		args := MakeArgs(s1, s2, s3)
		assert.Equal(t, len(args), len(s1)+len(s2)+len(s3))
	})

	t.Run("Any Arguments", func(t *testing.T) {
		addBase := 3
		args := MakeArgs(s1, s2, s3, "hello", 123, true)
		assert.Equal(t, len(args), len(s1)+len(s2)+len(s3)+addBase)
	})

	t.Run("Nil Arguments Only", func(t *testing.T) {
		addBase := 5
		args := MakeArgs(nil, nil, nil, nil, nil)
		assert.Equal(t, len(args), addBase)
	})
}

func TestMakeArgsPtr(t *testing.T) {
	s1 := []int{1, 2, 3, 4}

	args := MakeArgs(&s1)

	assert.Equal(t, len(args), len(s1))
}
