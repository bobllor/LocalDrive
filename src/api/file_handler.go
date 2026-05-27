package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/bobllor/gologger"
	"github.com/google/uuid"
)

const PARENT_ID_KEY = "parentID"
const FILE_ID_DOWNLOAD_KEY = "fileId"

var FileGetFileParentRoute = fmt.Sprintf("GET /api/storage/folder/{%s}", PARENT_ID_KEY)
var FilePostDownloadFileRoute = fmt.Sprintf("GET /api/download/file/{%s}", FILE_ID_DOWNLOAD_KEY)

const (
	FileGetFileRootRoute    = "GET /api/storage"
	FilePostUploadFileRoute = "POST /api/upload"
)

const (
	HEADER_UPLOAD_REQUEST_ID   = "Upload-Request-Id"
	HEADER_UPLOAD_FILE_SIZE    = "Upload-File-Size"
	HEADER_UPLOAD_TOTAL_CHUNKS = "Upload-Total-Chunks"
	HEADER_UPLOAD_CHUNK_INDEX  = "Upload-Chunk-Index"
)

const CHUNKS_DIR_NAME = "lcschunks"

type FileHandler struct {
	gateway *dbgateway.Gateway
	deps    *utils.Deps

	// uploadSession is a map of request IDs to a chunkInfo used to
	// hold the information with multi-chunk requests.
	//
	// This is used to write files to the disk from uploads.
	uploadSessionMap map[string]chunkInfo

	// reqMutexes is the map of mutexes of a given request ID. This will ensure
	// the locks will be on the specific request ID.
	reqMutexes map[string]*sync.Mutex
}

type chunkInfo struct {
	// TotalChunks is the number of chunks that is expected with the upload.
	TotalChunks int

	// Dir is the directory where the chunks are stored.
	Dir string

	// FileSize is the size of the file in bytes.
	FileSize int

	// FileName is the name of the file being uploaded.
	FileName string

	// FileExtension is the extension of the file. This is extracted from the last value of the FileName,
	// and if it doesn't exist, then it will use an empty string.
	FileExtension string

	// ChunkIndexes is a map of indexes used to track the chunks. The value
	// is the absolute path to the written chunk.
	ChunkIndexes map[int]string
}

func NewFileHandler(gw *dbgateway.Gateway, logger *gologger.Logger) *FileHandler {
	return &FileHandler{
		gateway:          gw,
		deps:             utils.NewDeps(logger),
		uploadSessionMap: make(map[string]chunkInfo),
	}
}

// DownloadFile handles downloading a file to the client using a stream on the
// ResponseWriter.
//
// This requires auth due to the use of an account ID.
func (fh *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	usrInfo, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.deps.Log.Critical("User context is not of type *dbgateway.UserSessionInfo")
		WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)
		return
	}

	fileId := r.PathValue(FILE_ID_DOWNLOAD_KEY)
	if fileId == "" {
		fh.deps.Log.Warnf("Path value %s is empty", FILE_ID_DOWNLOAD_KEY)
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	fh.deps.Log.Debugf("File ID: %s", fileId)

	fi, err := fh.gateway.File.GetFile(usrInfo.AccountId, fileId)
	if err != nil {
		fh.deps.Log.Criticalf("Error while attempting to query file: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	if fi == nil {
		fh.deps.Log.Warnf("File ID %s does not exist", fileId)
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonFileDoesNotExist)
		return
	}
	if fi.Type == file.FileTypeDir {
		// NOTE: this requires to create the files and zip it in the structure, recursively.
		// honestly it shouldnt be that hard to make but its not a priority.
		fh.deps.Log.Warnf("Directories are unsupported for downloading (file ID: %s)", fileId)
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	filePath := filepath.Join(fh.gateway.Dir.Storage, fi.Path)
	f, err := os.Open(filePath)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to open file: %v | Path: %s", err, fi.Path)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer f.Close()

	// removing leading periods for normalization
	fileExt := strings.TrimPrefix(fi.Extension, ".")
	fileName := fmt.Sprintf("%s.%s", fi.Name, fileExt)
	encodedFileName := url.PathEscape(fileName)
	dispostionValue := fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fileName, encodedFileName)

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set(ContentTypeKey, ContentOctet)
	w.Header().Set(ContentDispositionKey, dispostionValue)
	w.Header().Set("Access-Control-Expose-Headers", ContentDispositionKey)
	w.Header().Set("Content-Length", strconv.Itoa(int(fi.Size)))

	n, err := io.Copy(w, f)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to write file to stream: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	fh.deps.Log.Infof("Wrote %d bytes to stream", n)
}

// UploadFile uploads a file to the backend. The uploaded file can be sent in multiple chunks,
// and the handler is expected to be called multiple times.
// The chunks will be created in the temporary directory given in Gateway.Dir.Temp.
//
// The request header is expected to contain the request ID, metadata of the file information,
// the total chunks expected, and the 0-indexed current chunk.
// The response body is expected to be the bytes of the file uploaded.
//
// If the written chunks matches the total chunks, the handler will not continue. To complete
// the chunk upload, the complete upload endpoint must be used.
// Chunks with the same index will overwrite existing chunks of that index.
//
// This requires the auth middleware.
func (fh *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := CheckRequestHeaderNotEmpty(r, []string{
		HEADER_UPLOAD_FILE_SIZE,
		HEADER_UPLOAD_REQUEST_ID,
		HEADER_UPLOAD_TOTAL_CHUNKS,
		HEADER_UPLOAD_CHUNK_INDEX,
	})
	if err != nil {
		errMsg := utils.ToUpperFirstChar(err.Error())
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}

	reqId := r.Header.Get(HEADER_UPLOAD_REQUEST_ID)
	fileSize, err := strconv.Atoi(r.Header.Get(HEADER_UPLOAD_FILE_SIZE))
	if err != nil {
		errMsg := fmt.Sprintf("Header %s is not a number (got %s)", HEADER_UPLOAD_FILE_SIZE, r.Header.Get(HEADER_UPLOAD_FILE_SIZE))
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}
	totalChunks, err := strconv.Atoi(r.Header.Get(HEADER_UPLOAD_TOTAL_CHUNKS))
	if err != nil {
		errMsg := fmt.Sprintf("Header %s is not a number (got %s)", HEADER_UPLOAD_TOTAL_CHUNKS, r.Header.Get(HEADER_UPLOAD_TOTAL_CHUNKS))
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}
	chunkIndex, err := strconv.Atoi(r.Header.Get(HEADER_UPLOAD_CHUNK_INDEX))
	if err != nil {
		errMsg := fmt.Sprintf("Header %s is not a number (got %s)", HEADER_UPLOAD_CHUNK_INDEX, r.Header.Get(HEADER_UPLOAD_CHUNK_INDEX))
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}

	cinfo, ok := fh.uploadSessionMap[reqId]
	if !ok {
		cinfo = chunkInfo{
			TotalChunks:  totalChunks,
			ChunkIndexes: make(map[int]string),
			FileSize:     fileSize,
		}

		fh.deps.Log.Infof("New chunk info created for %s, total chunks: %d", reqId, totalChunks)
		fh.uploadSessionMap[reqId] = cinfo
	}

	_, err = os.Stat(cinfo.Dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fh.deps.Log.Criticalf("Failed to stat directory %s: %v", cinfo.Dir, err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	// if file does not exist or dir is empty
	if cinfo.Dir == "" || errors.Is(err, os.ErrNotExist) {
		fh.deps.Log.Infof("No chunks directory found for request %s", reqId)

		reqChunkDir := filepath.Join(fh.gateway.Dir.Temp, CHUNKS_DIR_NAME, reqId)
		err = os.MkdirAll(reqChunkDir, 0o700)
		if err != nil {
			fh.deps.Log.Criticalf("Failed to create directory: %v | Path: %s", err, reqChunkDir)
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}

		fh.deps.Log.Infof("Directory set to %s", reqChunkDir)
		cinfo.Dir = reqChunkDir
		// setting directory to the map
		fh.uploadSessionMap[reqId] = cinfo
	}

	chunkName := fmt.Sprintf("chunk-i0%d-*", chunkIndex)
	chunkFile, err := os.CreateTemp(cinfo.Dir, chunkName)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to create temporary chunk file: %v | Path: %s", err, cinfo.Dir)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer chunkFile.Close()

	n, err := io.Copy(chunkFile, r.Body)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to write response body to file: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	cinfo.ChunkIndexes[chunkIndex] = chunkFile.Name()
	fh.deps.Log.Infof("Wrote %d bytes to %s", n, chunkFile.Name())
}

// CompleteUploadFile is used to indicate that the uploading request has been completed.
// This requires the request ID to exist in the chunk map and all chunks must be accounted
// for.
//
// If the request ID does not exist, the total chunks doesn't match the written chunks,
// or missing/invalid headers, then the request will be rejected.
//
// This requires the auth middleware.
func (fh *FileHandler) CompleteUploadFile(w http.ResponseWriter, r *http.Request) {
	usr, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)
		return
	}

	err := CheckRequestHeaderNotEmpty(r, []string{HEADER_UPLOAD_REQUEST_ID})
	if err != nil {
		errMsg := utils.ToUpperFirstChar(err.Error())
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}

	reqId := r.Header.Get(HEADER_UPLOAD_REQUEST_ID)

	cinfo, ok := fh.uploadSessionMap[reqId]
	if !ok {
		errMsg := fmt.Sprintf("Request ID %s does not exist", reqId)
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}
	if cinfo.TotalChunks != len(cinfo.ChunkIndexes) {
		fh.deps.Log.Warnf("Total chunks %d does not match written chunks %d", cinfo.TotalChunks, len(cinfo.ChunkIndexes))
		WriteErrorResponse(w, "Missing chunks", http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	accountDir, err := fh.mkAccountDir(usr.AccountId)
	if err != nil {
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	fileId := uuid.NewString()
	storedFilePath := filepath.Join(accountDir, fileId)

	sf, err := os.Open(storedFilePath)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to open file: %v | Path: %s", err, storedFilePath)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer sf.Close()

	for i := range cinfo.TotalChunks {
		chunkPath, ok := cinfo.ChunkIndexes[i]
		if !ok {
			fh.deps.Log.Warnf("Missing chunk index %d", i)
			WriteErrorResponse(w, "Missing chunks", http.StatusBadRequest, ReasonBadRequestData)
			return
		}

		cf, err := os.Open(chunkPath)
		if err != nil {
			fh.deps.Log.Criticalf("Failed to open file: %v | Path: %s", err, chunkPath)
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}
		defer cf.Close()

		n, err := io.Copy(sf, cf)
		// abort the process, delete the main file
		if err != nil {
			fh.deps.Log.Criticalf("Failed to write to stored file: %v | Path: %s | Chunk index: %d", err, sf.Name(), i)

			err = os.Remove(sf.Name())
			if err != nil {
				fh.deps.Log.Criticalf("Failed to remove file: %v", err)
			} else {
				fh.deps.Log.Infof("Removed file %s", sf.Name())
			}

			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}

		fh.deps.Log.Infof("Wrote %d bytes to %s with chunk index %d", n, sf.Name(), i)
	}
}

// GetFiles retrieves a slice of Files based on the account ID and the given
// parent folder ID.
//
// If a parent folder ID is given, it will retrieve those parent folder files.
// If parent folder is nil, then it will retrieve the files with a nil parent or the
// root children.
//
// This requires the auth middleware wrapper due to the context.
func (fh *FileHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue(PARENT_ID_KEY)
	fh.deps.Log.Debugf("Request query: %v", parentID)

	userContext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.deps.Log.Critical("User context is not of type *dbgateway.UserSessionInfo")
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	if userContext == nil {
		requestContext, _ := GetRequestContext[string](r, CONTEXT_REQUEST_ID_KEY)
		fh.deps.Log.Infof("Unauthorized access | Request ID: %s", requestContext)
		WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	files, err := fh.gateway.File.GetFilesByAccountIdAndParentId(userContext.AccountId, parentID)
	if err == dbgateway.FileDoesNotExistErr {
		fh.deps.Log.Infof("Given file ID %s does not exist: %v", parentID, err)
		WriteErrorResponse(w, "Invalid file ID", http.StatusBadRequest, ReasonBadRequestData)
		return
	}
	if err != nil {
		fh.deps.Log.Criticalf("Failed to retrieve files with session ID and parent folder ID: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	convertedFiles := file.ToFileResponses(files...)
	res := NewApiResponse(convertedFiles)
	n, err := WriteResponse(w, res)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to write response to client: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	fh.deps.Log.Debugf("Response bytes: %d", n)
}

// mkAccountDir creates the directory of the account ID. It will return the directory path upon
// creation or if it already exists.
func (fh *FileHandler) mkAccountDir(accountId string) (string, error) {
	path := filepath.Join(fh.gateway.Dir.Storage, accountId)

	err := os.MkdirAll(path, 0o700)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to create account storage: %v | Path: %s", err, path)
		return "", err
	}

	return path, nil
}

// clearMutextEntry clears the entry of a given upload ID from the
// mutex map.
//
// If the request ID does not exist, then it will do nothing.
func (fh *FileHandler) clearMutexEntry(uploadId string) {
	_, ok := fh.reqMutexes[uploadId]
	if ok {
		delete(fh.reqMutexes, uploadId)
		fh.deps.Log.Infof("Cleared mutex for upload ID %s", uploadId)
	} else {
		fh.deps.Log.Infof("No mutex entry found for upload ID %s", uploadId)
	}
}
