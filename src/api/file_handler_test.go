package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/bobllor/assert"
	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/tests"
	"github.com/bobllor/cloud-project/src/utils"
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
		assert.Equal(t, output[1].FileID, tests.DbRowInfo.FileID)
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

func TestUploadGenerateId(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()
	mux.Handle(FilePostUploadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadGenerateId))

	serv := httptest.NewServer(mux)
	defer serv.Close()

	url := serv.URL + "/api/upload"
	tc := serv.Client()

	t.Run("Normal", func(t *testing.T) {
		fileName := "a video file.mp4"
		t.Cleanup(func() {
			deleteTestFile(t, ap, db, fileName)
		})

		body, err := tests.NewRequestBody(RequestFileUploadInfo{
			FileName:      fileName,
			FileSize:      12345555,
			FileParentId:  "",
			FileExtension: ".mp4",
			TotalChunks:   15,
		})
		assert.Nil(t, err)

		req, err := tests.NewRequest("POST", url, body)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := tc.Do(req)
		assert.Nil(t, err)
		defer res.Body.Close()

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

		req, err := tests.NewRequest("POST", url, body)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.True(t, res.StatusCode >= 400)
	})
}

func TestUploadFileChunk(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()
	mux.Handle(FilePostUploadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadGenerateId))
	mux.Handle(FilePostUploadFileChunkRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileChunk))

	serv := httptest.NewServer(mux)
	defer serv.Close()

	baseUrl := serv.URL
	tc := serv.Client()

	b := tests.GetBytes(4 * 1024)
	chunkLimit := float64(1024)
	chunks := math.Ceil(float64(len(b)) / chunkLimit)
	compareBytes := [][]byte{}

	fileName := "a text file.txt"
	t.Cleanup(func() {
		deleteTestFile(t, ap, db, fileName)
	})

	body, err := tests.NewRequestBody(RequestFileUploadInfo{
		FileName:      fileName,
		FileSize:      len(b),
		FileParentId:  "",
		FileExtension: ".txt",
		TotalChunks:   int(chunks),
	})
	assert.Nil(t, err)

	req, err := tests.NewRequest("POST", baseUrl+"/api/upload", body)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	uploadres, err := tc.Do(req)
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

		req.AddCookie(tests.GetCookie(CookieSessionKey))

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
	mux.Handle(FilePostUploadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadGenerateId))
	mux.Handle(FilePostUploadFileChunkRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileChunk))
	mux.Handle(FilePostUploadFileCompleteRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileComplete))

	serv := httptest.NewServer(mux)
	defer serv.Close()

	baseUrl := serv.URL
	tc := serv.Client()

	b := tests.GetBytes(4 * 1024)
	chunkLimit := float64(1024)
	chunks := math.Ceil(float64(len(b)) / chunkLimit)

	fileName := "a text file.txt"
	t.Cleanup(func() {
		deleteTestFile(t, ap, db, fileName)
	})
	body, err := tests.NewRequestBody(RequestFileUploadInfo{
		FileName:      fileName,
		FileSize:      len(b),
		FileParentId:  "",
		FileExtension: ".txt",
		TotalChunks:   int(chunks),
	})
	assert.Nil(t, err)

	req, err := tests.NewRequest("POST", baseUrl+"/api/upload", body)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	uploadres, err := tc.Do(req)
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
		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)

		start += int(chunkLimit)
		end += int(chunkLimit)
	}

	req, err = tests.NewRequest("POST", baseUrl+"/api/upload/"+idres.Output+"/complete", nil)
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

	fi, err := ap.gateway.File.GetFile(tests.DbRowInfo.AccountID, apres.Output.FileID)
	assert.Nil(t, err)
	assert.Equal(t, fi.Name, fileName)

	fiPath := filepath.Join(gw.Dir.Storage, fi.Path)

	stat, err := os.Stat(fiPath)
	assert.Nil(t, err)

	assert.Equal(t, stat.Name(), fi.FileID)
	assert.Equal(t, string(fi.UploadStatus), string(file.UploadCompleted))
}

func TestCompleteUploadErrorFile(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()
	mux.Handle(FilePostUploadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadGenerateId))
	mux.Handle(FilePostUploadFileChunkRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileChunk))
	mux.Handle(FilePostUploadFileCompleteRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileComplete))

	serv := httptest.NewServer(mux)
	defer serv.Close()

	baseUrl := serv.URL
	tc := serv.Client()

	b := tests.GetBytes(4 * 1024)
	chunkLimit := float64(1024)
	chunks := math.Ceil(float64(len(b)) / chunkLimit)

	fileName := "a text file.txt"
	t.Cleanup(func() {
		deleteTestFile(t, ap, db, fileName)
	})
	body, err := tests.NewRequestBody(RequestFileUploadInfo{
		FileName:      fileName,
		FileSize:      len(b),
		FileParentId:  "",
		FileExtension: ".txt",
		TotalChunks:   int(chunks),
	})
	assert.Nil(t, err)

	req, err := tests.NewRequest("POST", baseUrl+"/api/upload", body)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	uploadres, err := tc.Do(req)
	assert.Nil(t, err)
	assert.True(t, uploadres.StatusCode <= 300)
	defer uploadres.Body.Close()

	var idres ApiResponse[string]
	err = json.NewDecoder(uploadres.Body).Decode(&idres)
	assert.Nil(t, err)
	assert.Nil(t, idres.Error)

	cinfo, ok := ap.FileHandler.uploadSessions[idres.Output]
	assert.True(t, ok)

	t.Cleanup(func() {
		dbgateway.DropRows(db, file.TableName, file.ColumnFileID, cinfo.FileInfo.FileID)
	})

	start := 0
	end := int(chunkLimit)
	for i := range int(chunks) {
		if end > len(b) {
			end = len(b)
		}
		body := bytes.NewBuffer(b[start:end])
		req, err := http.NewRequest("POST", baseUrl+"/api/upload/"+idres.Output+"/"+strconv.Itoa(i), body)
		assert.Nil(t, err)
		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := tc.Do(req)
		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)

		start += int(chunkLimit)
		end += int(chunkLimit)
	}

	req, err = tests.NewRequest("POST", baseUrl+"/api/upload/"+idres.Output+"/complete", nil)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	osCreateOriginal := osCreate
	defer func() { osCreate = osCreateOriginal }()
	osCreate = func(name string) (*os.File, error) {
		return nil, fmt.Errorf("Mock error")
	}

	res, err := tc.Do(req)
	assert.Nil(t, err)
	defer res.Body.Close()

	var apires ApiResponse[any]
	err = json.NewDecoder(res.Body).Decode(&apires)
	assert.Nil(t, err)
	assert.NotNil(t, apires.Error)

	// fresh update
	fi, err := gw.File.GetFile(cinfo.FileInfo.OwnerID, cinfo.FileInfo.FileID)
	assert.Nil(t, err)

	assert.Equal(t, string(cinfo.FileInfo.UploadStatus), string(file.UploadPending))
	assert.Equal(t, string(fi.UploadStatus), string(file.UploadFailed))
}

func TestAddFolder(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t)
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()

	mux.Handle(FilePostAddFolderRoute, ap.CreateAuthMiddleware(ap.FileHandler.PostAddFolder))
	serv := httptest.NewServer(mux)

	tc := serv.Client()
	defer serv.Close()

	folderName := "very secret folder"
	reqData := RequestAddFolderInfo{
		Name:     folderName,
		ParentId: folderName,
	}
	reqBody, err := json.Marshal(reqData)
	assert.Nil(t, err)

	req, err := tests.NewRequest("POST", serv.URL+"/api/folders/add", bytes.NewBuffer(reqBody))
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	res, err := tc.Do(req)
	assert.Nil(t, err)
	assert.True(t, res.StatusCode < 400)
	defer res.Body.Close()

	var apires ApiResponse[file.FileResponse]
	err = json.NewDecoder(res.Body).Decode(&apires)
	assert.Nil(t, err)
	assert.Equal(t, apires.Status, StatusSuccess)

	fileRes := apires.Output
	t.Cleanup(func() {
		dbgateway.DropRows(db, file.TableName, file.ColumnFileID, fileRes.FileID)
	})

	af, err := ap.gateway.File.GetFile(tests.DbRowInfo.AccountID, fileRes.FileID)
	assert.Nil(t, err)
	assert.NotNil(t, af)
	assert.TrueAll(t, af.Name == fileRes.Name, af.FileID == fileRes.FileID, af.Type == fileRes.Type)
}

func TestUpdateStatusFailSuccess(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t, dbgateway.GatewayDBOptions{CreateTemp: true})
	ap := NewApiHandler(gw, tests.NewTestLogger())

	serv := newTestServer(gw)
	defer serv.Close()

	tc := serv.Client()
	fileName := "video.mp4"
	t.Cleanup(func() {
		deleteTestFile(t, ap, db, fileName)
	})

	body, err := tests.NewRequestBody(RequestFileUploadInfo{
		FileName:      fileName,
		FileSize:      12345555,
		FileParentId:  "",
		FileExtension: ".mp4",
		TotalChunks:   15,
	})
	assert.Nil(t, err)

	req, err := tests.NewRequest("POST", serv.URL+"/api/upload", body)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	res, err := tc.Do(req)
	assert.Nil(t, err)
	defer res.Body.Close()

	var idres ApiResponse[string]
	err = json.NewDecoder(res.Body).Decode(&idres)
	assert.Nil(t, err)

	req, err = tests.NewRequest("PATCH", serv.URL+"/api/upload/"+idres.Output+"/fail", nil)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	res, err = tc.Do(req)
	assert.Nil(t, err)
	defer res.Body.Close()

	var upres ApiResponse[bool]
	err = json.NewDecoder(res.Body).Decode(&upres)
	assert.Nil(t, upres.Error)
	assert.True(t, upres.Output)

	files, err := gw.File.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	var genFile *file.FileResponse
	for _, fi := range files {
		if fi.Name == fileName {
			genFile = &fi
			break
		}
	}

	assert.NotNil(t, genFile)
	assert.Equal(t, string(genFile.UploadStatus), string(file.UploadFailed))
}

func TestGetFolderBreadcrumbs(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t)

	f1 := file.NewFile(tests.DbRowInfo.AccountID, "folder2",
		file.FileTypeDir, "", 0, tests.DbRowInfo.ParentID, file.UploadCompleted)
	f2 := file.NewFile(tests.DbRowInfo.AccountID, "folder3",
		file.FileTypeDir, "", 0, f1.FileID, file.UploadCompleted)

	files := []file.File{f1, f2}

	err := gw.File.AddFile(files...)
	assert.Nil(t, err)

	t.Cleanup(func() {
		for _, f := range files {
			dbgateway.DropRows(db, file.TableName, file.ColumnFileID, f.FileID)
		}
	})

	serv := newTestServer(gw)
	tc := serv.Client()

	t.Run("Normal run", func(t *testing.T) {
		req, err := tests.NewRequest("GET", serv.URL+"/api/folders/"+f2.FileID+"/breadcrumbs", nil)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := tc.Do(req)
		assert.Nil(t, err)
		defer res.Body.Close()

		var apres ApiResponse[[]dbgateway.BreadcrumbFile]
		err = json.NewDecoder(res.Body).Decode(&apres)
		assert.Nil(t, err)
		assert.NotNil(t, apres.Output)

		assert.Equal(t, len(apres.Output), 3)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		req, err := tests.NewRequest("GET", serv.URL+"/api/folders/"+"abcdefg1234"+"/breadcrumbs", nil)
		assert.Nil(t, err)

		req.AddCookie(tests.GetCookie(CookieSessionKey))

		res, err := tc.Do(req)

		assert.Nil(t, err)
		assert.Equal(t, res.StatusCode, http.StatusBadRequest)
	})
}

func TestRenameFile(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t)
	serv := newTestServer(gw)
	defer serv.Close()

	tc := serv.Client()
	url := serv.URL + "/api/file/rename"

	type testCases struct {
		File            file.File
		NewName         string
		TestName        string
		OverwriteFileId string
		NilReqBody      bool
		IsErr           bool
		ErrReasonCode   ReasonCode
		StatusCode      int
	}

	cases := []testCases{
		{
			File: file.NewFile(
				tests.DbRowInfo.AccountID, "example1", file.FileTypeFile,
				".txt", 0, "", file.UploadCompleted,
			),
			NewName:  "new name 1",
			TestName: "Normal run",
			IsErr:    false,
		},
		{
			File: file.NewFile(
				tests.DbRowInfo.AccountID, "example2", file.FileTypeFile,
				".txt", 0, "", file.UploadCompleted,
			),
			NewName:       tests.DbRowInfo.FileName,
			TestName:      "Duplicate error",
			IsErr:         true,
			ErrReasonCode: ReasonDuplicateData,
			StatusCode:    http.StatusBadRequest,
		},
		{
			File: file.NewFile(
				tests.DbRowInfo.AccountID, "example3", file.FileTypeFile,
				".txt", 0, "", file.UploadCompleted,
			),
			NewName:       "",
			TestName:      "Invalid request body: empty name",
			IsErr:         true,
			ErrReasonCode: ReasonBadRequestData,
			StatusCode:    http.StatusBadRequest,
		},
		{
			File: file.NewFile(
				tests.DbRowInfo.AccountID, "example4", file.FileTypeFile,
				".txt", 0, "", file.UploadCompleted,
			),
			NewName:         "file name",
			TestName:        "Invalid request body: invalid file ID",
			IsErr:           true,
			OverwriteFileId: "asdfdsa-12312",
			ErrReasonCode:   ReasonBadRequestData,
			StatusCode:      http.StatusBadRequest,
		},
		{
			File: file.NewFile(
				tests.DbRowInfo.AccountID, "example4", file.FileTypeFile,
				".txt", 0, "", file.UploadCompleted,
			),
			NewName:       "file name",
			TestName:      "Empty request body",
			NilReqBody:    true,
			IsErr:         true,
			ErrReasonCode: ReasonBadRequestData,
			StatusCode:    http.StatusBadRequest,
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			err := gw.File.AddFile(c.File)
			assert.Nil(t, err)

			t.Cleanup(func() {
				dbgateway.DropRows(db, file.TableName, file.ColumnFileID, c.File.FileID)
			})

			body := RequestRenameFile{FileId: c.File.FileID, NewFileName: c.NewName}
			if c.OverwriteFileId != "" {
				body.FileId = c.OverwriteFileId
			}

			reqBody, err := tests.NewRequestBody(body)
			assert.Nil(t, err)

			if c.NilReqBody {
				reqBody = nil
			}

			req, err := tests.NewRequest("PATCH", url, reqBody)
			assert.Nil(t, err)

			req.AddCookie(tests.GetCookie(CookieSessionKey))

			res, err := tc.Do(req)
			assert.Nil(t, err)
			defer res.Body.Close()

			resp, err := utils.Decode[ApiResponse[file.FileResponse]](res.Body)
			assert.Nil(t, err)
			if !c.IsErr {
				assert.Nil(t, resp.Error)

				// name checks
				fires := resp.Output
				assert.Equal(t, fires.Name, c.NewName)
				fi, err := gw.File.GetFile(tests.DbRowInfo.AccountID, c.File.FileID)
				assert.Nil(t, err)
				assert.Equal(t, fi.Name, c.NewName)
			} else {
				assert.NotNil(t, resp.Error)
				assert.Equal(t, resp.Error.Reason, c.ErrReasonCode)
				assert.Equal(t, resp.Error.Code, c.StatusCode)
			}
		})
	}
}

func TestGetDeletedFiles(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t)

	fname := "deletedfilename"
	dname := "deleteddirname"
	df := file.NewFile(tests.DbRowInfo.AccountID, fname, file.FileTypeFile, "", 0, "", file.UploadCompleted)
	dd := file.NewFile(tests.DbRowInfo.AccountID, dname, file.FileTypeDir, "", 0, "", file.UploadCompleted)
	now := time.Now()

	df.DeletedOn = &now
	dd.DeletedOn = &now

	err := gw.File.AddFile(df, dd)
	assert.Nil(t, err)

	t.Cleanup(func() {
		dbgateway.DropRows(db, file.TableName, file.ColumnFileID, df.FileID, dd.FileID)
	})

	serv := newTestServer(gw)
	defer serv.Close()

	url := serv.URL + "/api/storage/trash"
	tc := serv.Client()

	req, err := tests.NewRequest("GET", url, nil)
	assert.Nil(t, err)

	req.AddCookie(tests.GetCookie(CookieSessionKey))

	res, err := tc.Do(req)
	assert.Nil(t, err)

	defer res.Body.Close()

	var apires ApiResponse[[]file.FileResponse]
	err = json.NewDecoder(res.Body).Decode(&apires)
	assert.Nil(t, err)

	files := apires.Output
	assert.Equal(t, apires.Status, StatusSuccess)
	assert.Equal(t, len(files), 2)

	// sorting check
	assert.Equal(t, files[0].FileID, dd.FileID)
	assert.Equal(t, files[1].FileID, df.FileID)
}

func TestDeleteFile(t *testing.T) {
	gw, db := dbgateway.NewTestGatewayDB(t)

	serv := newTestServer(gw)
	cases := []struct {
		name      string
		addFile   bool
		isSuccess bool
	}{
		{
			name:      "Successful deletion",
			addFile:   true,
			isSuccess: true,
		},
		{
			name: "Failed deletion",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fi := file.NewFile(tests.DbRowInfo.AccountID, c.name, file.FileTypeFile, ".txt", 0, "", file.UploadCompleted)

			if c.addFile {
				t.Cleanup(func() {
					dbgateway.DropRows(db, file.TableName, file.ColumnFileID, fi.FileID)
				})

				err := gw.File.AddFile(fi)
				assert.Nil(t, err)
			}

			url := serv.URL + "/api/file/delete/" + fi.FileID

			req, err := tests.NewRequest("DELETE", url, nil)
			assert.Nil(t, err)

			req.AddCookie(tests.GetCookie(CookieSessionKey))

			tc := serv.Client()

			res, err := tc.Do(req)
			assert.Nil(t, err)

			apires, err := utils.Decode[*ApiResponse[bool]](res.Body)
			assert.Nil(t, err)

			if c.isSuccess {
				assert.Equal(t, apires.Status, StatusSuccess)
				assert.True(t, apires.Output)

				sfi, err := gw.File.GetFile(tests.DbRowInfo.AccountID, fi.FileID)
				assert.Nil(t, err)

				assert.NotNil(t, sfi)
				assert.Equal(t, sfi.FileID, fi.FileID)
			} else {
				assert.Equal(t, apires.Status, StatusSuccess)

				assert.False(t, apires.Output)

				assert.NotNil(t, apires.Error)
				assert.Equal(t, apires.Error.Code, http.StatusNotFound)
				assert.Equal(t, apires.Error.Reason, ReasonNotFound)
			}
		})
	}
}

// newTestServer creates a new test HTTP server with all the
// default routes and functions from the File handler. It will automatically
// wrap handlers in middleware.
func newTestServer(gw *dbgateway.Gateway) *httptest.Server {
	ap := NewApiHandler(gw, tests.NewTestLogger())

	mux := http.NewServeMux()

	mux.Handle(FilePostUploadFileChunkRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileChunk))
	mux.Handle(FilePostUploadFileCompleteRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileComplete))
	mux.Handle(FilePostUploadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadGenerateId))
	mux.Handle(FilePatchUpdateFileStatusRoute, ap.CreateAuthMiddleware(ap.FileHandler.UploadFileStatusFailed))
	mux.Handle(FileGetFileRootRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetFiles))
	mux.Handle(FileGetFileParentRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetFiles))
	mux.Handle(FilePostDownloadFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.DownloadFile))
	mux.Handle(FilePostAddFolderRoute, ap.CreateAuthMiddleware(ap.FileHandler.PostAddFolder))
	mux.Handle(FileGetFolderBreadcrumbsRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetFolderBreadcrumbs))
	mux.Handle(FilePatchRenameFileRoute, ap.CreateAuthMiddleware(ap.FileHandler.RenameFile))
	mux.Handle(FileGetDeletedFilesRoute, ap.CreateAuthMiddleware(ap.FileHandler.GetDeletedFiles))
	mux.Handle(FileDeleteFileDeleteionRoute, ap.CreateAuthMiddleware(ap.FileHandler.DeleteFile))

	serv := httptest.NewServer(mux)

	return serv
}

// deleteTestFile deletes a given file name from the File database of the default
// test account.
func deleteTestFile(t *testing.T, ap *ApiHandler, db *sql.DB, fileName string) {
	files, err := ap.gateway.File.GetAllFiles(tests.DbRowInfo.AccountID)
	assert.Nil(t, err)

	for _, fi := range files {
		if fi.Name == fileName {
			dbgateway.DropRows(db, file.TableName, file.ColumnFileID, fi.FileID)
		}
	}
}
