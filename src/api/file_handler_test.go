package api

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

		var apiRes ApiResponse[[]file.FileResponse]
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

		var apiRes ApiResponse[[]file.FileResponse]
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

		var apiRes ApiResponse[[]file.FileResponse]
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

		var apiRes ApiResponse[any]
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

		var apiRes ApiResponse[any]
		err = json.NewDecoder(res.Body).Decode(&apiRes)
		assert.Nil(t, err)

		assert.NotNil(t, apiRes.Error)
		assert.Equal(t, apiRes.Error.Code, http.StatusBadRequest)
	})
}

func TestUploadFileId(t *testing.T) {
	gw, _ := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()
	mux.HandleFunc(FilePostUploadFileRoute, ap.FileHandler.UploadGenerateId)

	serv := httptest.NewServer(mux)
	defer serv.Close()

	url := serv.URL + "/api/upload"
	tc := serv.Client()

	t.Run("Normal", func(t *testing.T) {
		fileName := "a video file.mp4"
		body, err := tests.NewRequestBody(RequestFileUploadInfo{
			FileName:      fileName,
			FileSize:      12345555,
			FileParentId:  "",
			FileExtension: ".mp4",
			TotalChunks:   15,
		})
		assert.Nil(t, err)

		res, err := tc.Post(url, ContentJson, body)
		assert.Nil(t, err)

		var apres ApiResponse[string]
		err = json.NewDecoder(res.Body).Decode(&apres)
		assert.Nil(t, err)
		assert.Nil(t, apres.Error)

		_, ok := ap.FileHandler.uploadSessions[apres.Output]
		assert.True(t, ok)
	})

	t.Run("Invalid body", func(t *testing.T) {
		body, err := tests.NewRequestBody(ApiResponse[any]{})
		assert.Nil(t, err)

		res, err := tc.Post(url, ContentJson, body)
		assert.Nil(t, err)
		assert.True(t, res.StatusCode >= 400)
	})
}

func TestUploadFileChunk(t *testing.T) {
	gw, _ := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()
	mux.HandleFunc(FilePostUploadFileRoute, ap.FileHandler.UploadGenerateId)
	mux.HandleFunc(FilePostUploadFileChunkRoute, ap.FileHandler.UploadFileChunk)

	serv := httptest.NewServer(mux)
	defer serv.Close()

	baseUrl := serv.URL
	tc := serv.Client()

	b := tests.GetBytes(4 * 1024)
	chunkLimit := float64(1024)
	chunks := math.Ceil(float64(len(b)) / chunkLimit)
	compareBytes := [][]byte{}

	fileName := "a text file.txt"
	body, err := tests.NewRequestBody(RequestFileUploadInfo{
		FileName:      fileName,
		FileSize:      len(b),
		FileParentId:  "",
		FileExtension: ".txt",
		TotalChunks:   int(chunks),
	})
	assert.Nil(t, err)

	uploadres, err := tc.Post(baseUrl+"/api/upload", ContentJson, body)
	assert.Nil(t, err)
	assert.True(t, uploadres.StatusCode <= 300)
	defer uploadres.Body.Close()

	var idres ApiResponse[string]
	err = json.NewDecoder(uploadres.Body).Decode(&idres)
	assert.Nil(t, err)
	assert.Nil(t, idres.Error)

	start := 0
	end := int(chunkLimit)
	for i := range int(chunks) {
		if end > len(b) {
			end = len(b)
		}
		compareBytes = append(compareBytes, b[start:end])
		body := bytes.NewBuffer(b[start:end])
		req, err := http.NewRequest("POST", baseUrl+"/api/upload/"+idres.Output+"/"+strconv.Itoa(i), body)
		assert.Nil(t, err)

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)

		start += int(chunkLimit)
		end += int(chunkLimit)
	}

	uploadMap := ap.FileHandler.uploadSessions
	reqMap, ok := uploadMap[idres.Output]
	assert.True(t, ok)
	assert.Equal(t, len(reqMap.ChunkIndexes), int(chunks))

	for i := range int(chunks) {
		chunkPath, ok := reqMap.ChunkIndexes[i]
		assert.True(t, ok)

		p := filepath.Join(chunkPath)
		cb, err := os.ReadFile(p)
		assert.Nil(t, err)

		// this assumes that the file maintains order.. but i wrote the file name
		// creation to maintain this order
		baseBytes := compareBytes[i]

		assert.Equal(t, string(baseBytes), string(cb))
	}
}

func TestCompleteUploadFile(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()
	mux.HandleFunc(FilePostUploadFileRoute, ap.FileHandler.UploadGenerateId)
	mux.HandleFunc(FilePostUploadFileChunkRoute, ap.FileHandler.UploadFileChunk)
	mux.Handle(FilePostUploadFileCompleteRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileComplete))

	serv := httptest.NewServer(mux)
	defer serv.Close()

	baseUrl := serv.URL
	tc := serv.Client()

	b := tests.GetBytes(4 * 1024)
	chunkLimit := float64(1024)
	chunks := math.Ceil(float64(len(b)) / chunkLimit)

	fileName := "a text file.txt"
	body, err := tests.NewRequestBody(RequestFileUploadInfo{
		FileName:      fileName,
		FileSize:      len(b),
		FileParentId:  "",
		FileExtension: ".txt",
		TotalChunks:   int(chunks),
	})
	assert.Nil(t, err)

	uploadres, err := tc.Post(baseUrl+"/api/upload", ContentJson, body)
	assert.Nil(t, err)
	assert.True(t, uploadres.StatusCode <= 300)
	defer uploadres.Body.Close()

	var idres ApiResponse[string]
	err = json.NewDecoder(uploadres.Body).Decode(&idres)
	assert.Nil(t, err)
	assert.Nil(t, idres.Error)

	start := 0
	end := int(chunkLimit)
	for i := range int(chunks) {
		if end > len(b) {
			end = len(b)
		}
		body := bytes.NewBuffer(b[start:end])
		req, err := http.NewRequest("POST", baseUrl+"/api/upload/"+idres.Output+"/"+strconv.Itoa(i), body)
		assert.Nil(t, err)

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)

		start += int(chunkLimit)
		end += int(chunkLimit)
	}

	req, err := tests.NewRequest("POST", baseUrl+"/api/upload/"+idres.Output+"/complete", nil)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	res, err := tc.Do(req)
	assert.Nil(t, err)
	assert.True(t, res.StatusCode < 300)
	defer res.Body.Close()

	var apres ApiResponse[*file.FileResponse]
	err = json.NewDecoder(res.Body).Decode(&apres)
	assert.Nil(t, err)
	assert.Nil(t, apres.Error)

	t.Cleanup(func() {
		dbgateway.DropRows(db, file.TableName, file.ColumnFileID, apres.Output.FileID)
	})
}
