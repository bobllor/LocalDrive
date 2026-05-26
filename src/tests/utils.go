package tests

import (
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobllor/gologger"
)

// NewTestLogger creates a new test logger with a silent output.
func NewTestLogger() *gologger.Logger {
	printer := log.New(os.Stdout, "", log.Ldate|log.Ltime)
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

// GetBytes creates a byte slice of arbitrary random data.
func GetBytes(size int) []byte {
	b := make([]byte, 0, size)
	r := rand.New(rand.NewSource(time.Now().Unix()))

	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	chars += strings.ToLower(chars)

	for range size {
		ranCh := r.Intn(len(chars) - 1)
		b = append(b, chars[ranCh])
	}

	return b
}
