package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bobllor/assert"
	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/tests"
)

func TestGetFilesByAccountAndParent(t *testing.T) {
	mux := http.NewServeMux()
	gw, _ := dbgateway.NewTestGatewayDB(t)
	ap := NewApiHandler(gw, tests.NewTestLogger())

	fh := NewFileHandler(gw, tests.NewTestLogger())

	mux.Handle(FileGetFileRootRoute, ap.CreateAuthMiddleware(fh.GetFiles))
	mux.Handle(FileGetFileParentRoute, ap.CreateAuthMiddleware(fh.GetFiles))
	tsv := httptest.NewServer(mux)
	defer tsv.Close()

	tc := tsv.Client()

	cookie := &http.Cookie{
		Name:  CookieSessionKey,
		Value: tests.DbRowInfo.SessionID,
	}

	t.Run("Root files", func(t *testing.T) {
		req, err := http.NewRequest("GET", tsv.URL+"/api/storage", bytes.NewBuffer([]byte{}))
		assert.Nil(t, err)

		req.AddCookie(cookie)

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.NotEqual(t, res.StatusCode, 404)
		defer res.Body.Close()

		var apiRes ApiResponse
		err = json.NewDecoder(res.Body).Decode(&apiRes)
		assert.Nil(t, err)
		assert.NotEqual(t, apiRes.Status, StatusError)

		var output []file.FileResponse
		d, err := json.Marshal(apiRes.Output)
		assert.Nil(t, err)

		err = json.Unmarshal(d, &output)
		assert.Nil(t, err)

		assert.Equal(t, len(output), 2)
		assert.Equal(t, output[0].FileID, tests.DbRowInfo.FileID)
	})

	t.Run("Child files from folder", func(t *testing.T) {
		// obtained from sql script in sql test db
		folder := "randomfolderidhere"
		req, err := http.NewRequest("GET", tsv.URL+"/api/storage/folder/"+folder, bytes.NewBuffer([]byte{}))
		assert.Nil(t, err)

		req.AddCookie(cookie)

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.NotEqual(t, res.StatusCode, 404)
		defer res.Body.Close()

		var apiRes ApiResponse
		err = json.NewDecoder(res.Body).Decode(&apiRes)
		assert.Nil(t, err)
		assert.NotEqual(t, apiRes.Status, StatusError)

		var output []file.FileResponse
		d, err := json.Marshal(apiRes.Output)
		assert.Nil(t, err)

		err = json.Unmarshal(d, &output)
		assert.Nil(t, err)

		assert.Equal(t, len(output), 1)
		// this is the child file, which has a different ID
		assert.Equal(t, output[0].FileID, "anotherfileidhere")
	})

	t.Run("Invalid folder", func(t *testing.T) {
		folder := "nonexistentidhere"
		req, err := http.NewRequest("GET", tsv.URL+"/api/storage/folder/"+folder, bytes.NewBuffer([]byte{}))
		assert.Nil(t, err)

		req.AddCookie(cookie)

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.True(t, res.StatusCode == http.StatusBadRequest)
		defer res.Body.Close()

		var apiRes ApiResponse
		err = json.NewDecoder(res.Body).Decode(&apiRes)
		assert.Nil(t, err)
		assert.Equal(t, apiRes.Status, StatusError)

		assert.Contains(t, apiRes.Error.Message, "Invalid")
	})
}

func TestDownloadFile(t *testing.T) {
	mux := http.NewServeMux()
	gw, _ := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateStorage: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux.Handle(FilePostDownloadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.DownloadFile))

	serv := httptest.NewServer(mux)
	defer serv.Close()

	url := serv.URL
	client := serv.Client()
	apiUrlNoFile := url + "/api/download/file/"

	t.Run("Normal process", func(t *testing.T) {
		req, err := tests.NewRequest("GET", apiUrlNoFile+tests.DbRowInfo.FileID, nil)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := client.Do(req)
		assert.Nil(t, err)
		defer res.Body.Close()

		_, params, err := mime.ParseMediaType(res.Header.Get(ContentDispositionKey))
		assert.Nil(t, err)
		fileName, ok := params["filename"]
		assert.True(t, ok)
		filePath := filepath.Join(t.TempDir(), fileName)
		f, err := os.Create(filePath)
		assert.Nil(t, err)
		defer f.Close()

		_, err = io.Copy(f, res.Body)
		assert.Nil(t, err)

		fs, err := os.Stat(filePath)
		assert.Nil(t, err)

		assert.Equal(t, fs.Name(), fileName)
	})

	t.Run("Invalid file ID", func(t *testing.T) {
		req, err := tests.NewRequest("GET", apiUrlNoFile+"fdsa1234dzlk2039sclkorsv", nil)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := client.Do(req)
		assert.Nil(t, err)
		defer res.Body.Close()

		var apiRes ApiResponse
		err = json.NewDecoder(res.Body).Decode(&apiRes)
		assert.Nil(t, err)

		assert.NotNil(t, apiRes.Error)
		assert.Equal(t, apiRes.Error.Reason, ReasonFileDoesNotExist)
	})

	t.Run("Directory file ID", func(t *testing.T) {
		// from test sql script
		req, err := tests.NewRequest("GET", apiUrlNoFile+"randomfolderidhere", nil)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := client.Do(req)
		assert.Nil(t, err)
		defer res.Body.Close()

		var apiRes ApiResponse
		err = json.NewDecoder(res.Body).Decode(&apiRes)
		assert.Nil(t, err)

		assert.NotNil(t, apiRes.Error)
		assert.Equal(t, apiRes.Error.Code, http.StatusBadRequest)
	})
}
