package tests

import (
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/bobllor/gologger"
)

type TestDbMeta struct {
	User     string
	Addr     string
	Password string
	Net      string
	DbName   string
}

// DbMetaInfo is a read-only struct used to hold information
// of the test database.
var DbMetaInfo = TestDbMeta{
	User:     "root",
	Addr:     ":3307",
	Password: "",
	Net:      "tcp",
	DbName:   "TestLocalCloudStorage",
}

type TestDbRow struct {
	AccountID  string
	Username   string
	UserActive bool
	PhcString  string
	SessionID  string
	FileID     string
	FileName   string
}

// DbRowInfo is a read-only variable that contains the default values
// included in the test database.
var DbRowInfo = TestDbRow{
	AccountID:  "89672a64-f3ff-490c-8f2d-7e5cf5d4aa70",
	Username:   "test.username",
	UserActive: true,
	PhcString:  "$argon2id$v=19$m=65536,t=2,p=4$QTdpUkJ3c3J0amlOT2huV2VBR2duZw$vzICl8p5CVfpGfypDV4yIVULsYatAmir6B8nHWtcPtE",
	SessionID:  "7ca90f85-b1e0-4214-8ff6-4e3720cc8078",
	FileID:     "randomfileidhere",
	FileName:   "test1.txt",
}

// TestPassword is the test password used to create the PhcString for
// the default entry in the test database.
var TestPassword = "anothertestpassword"

// TestSalt is the salt used to salt the test password for the
// default entry in the test database.
var TestSalt = []byte("A7iRBwsrtjiNOhnWeAGgng")

// NewTestLogger creates a new test logger with a silent output.
func NewTestLogger() *gologger.Logger {
	printer := log.New(io.Discard, "", log.Ldate|log.Ltime)
	log := gologger.NewLogger(printer, gologger.Lsilent)

	return log
}

// CreateFiles creates random files in the root. It will return
// a string slice containing the paths of the files created.
//
// root is the path where the files will be created in.
//
// An error will be returned if any errors occur during
// the file creation.
func CreateFiles(root string) ([]string, error) {
	paths := []string{}

	files := []string{
		"text1.txt",
		"text2.txt",
		"text3.txt",
		"some.logs.log",
		"database.db",
	}

	f1 := "folder1"
	f2 := "folder2"

	dirPaths := []string{
		root,
		filepath.Join(root, f1),
		filepath.Join(root, f2),
		filepath.Join(root, f1, f2),
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, file := range files {
		fullPath := filepath.Join(dirPaths[r.Intn(len(dirPaths)-1)], file)
		basePath := path.Dir(fullPath)

		err := os.MkdirAll(basePath, 0o744)
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(fullPath, []byte{}, 0o644)
		if err != nil {
			return nil, err
		}

		paths = append(paths, fullPath)
	}

	return paths, nil
}
