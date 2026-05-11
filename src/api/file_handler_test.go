package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bobllor/assert"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/tests"
)

func TestGetFilesBySessionAndParent(t *testing.T) {
	mux := http.NewServeMux()
	gw, _ := getGatewayDb(t)

	fh := NewFileHandler(gw, tests.NewTestLogger())

	mux.HandleFunc(FileGetFileRootRoute, fh.GetFiles)
	mux.HandleFunc(FileGetFileParentRoute, fh.GetFiles)
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

		var output []file.File
		d, err := json.Marshal(apiRes.Output)
		assert.Nil(t, err)

		err = json.Unmarshal(d, &output)
		assert.Nil(t, err)

		assert.Equal(t, len(output), 2)
		assert.Equal(t, output[0].OwnerID, tests.DbRowInfo.AccountID)
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

		var output []file.File
		d, err := json.Marshal(apiRes.Output)
		assert.Nil(t, err)

		err = json.Unmarshal(d, &output)
		assert.Nil(t, err)

		assert.Equal(t, len(output), 1)
		assert.Equal(t, output[0].OwnerID, tests.DbRowInfo.AccountID)
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
