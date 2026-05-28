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
	"time"

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
	FileGetFileRootRoute            = "GET /api/storage"
	FilePostUploadFileRoute         = "POST /api/upload"
	FilePostCompleteUploadFileRoute = "POST /api/upload/complete"
)

const (
	HEADER_UPLOAD_REQUEST_ID   = "Upload-Request-Id"
	HEADER_UPLOAD_FILE_SIZE    = "Upload-File-Size"
	HEADER_UPLOAD_TOTAL_CHUNKS = "Upload-Total-Chunks"
	HEADER_UPLOAD_CHUNK_INDEX  = "Upload-Chunk-Index"
	HEADER_UPLOAD_PARENT_ID    = "Upload-Parent-Id"
)

const CHUNKS_DIR_NAME = "lcschunks"

// FileHandler is the handler used for file related operations in the API.
type FileHandler struct {
	gateway *dbgateway.Gateway
	util    *Utility

	// uploadSession is a map of request IDs to a chunkInfo used to
	// hold the information with multi-chunk requests.
	//
	// This is used to write files to the disk from uploads.
	uploadSessions map[string]chunkInfo

	// uploadMutexes is the map of mutexes of a given upload ID for uploading files. This ensures that
	// chunks will not be modified or accessed while one is already active.
	uploadMutexes map[string]*sync.Mutex

	// completeMutexes is the map of mutexes of a given upload ID for completion of the uploading files.
	// This prevents overwriting or accessing the same file during chunk merging.
	completeMutexes map[string]*sync.Mutex
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
		gateway:         gw,
		util:            &Utility{Log: logger},
		uploadSessions:  make(map[string]chunkInfo),
		uploadMutexes:   make(map[string]*sync.Mutex),
		completeMutexes: make(map[string]*sync.Mutex),
	}
}

// DownloadFile handles downloading a file to the client using a stream on the
// ResponseWriter.
//
// This requires auth due to the use of an account ID.
func (fh *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	usrInfo, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.Log.Critical("User context is not of type *dbgateway.UserSessionInfo")
		WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)
		return
	}

	fileId := r.PathValue(FILE_ID_DOWNLOAD_KEY)
	if fileId == "" {
		fh.util.HttpWriteBadDataError(w, "Path value %s is empty", FILE_ID_DOWNLOAD_KEY)
		return
	}

	fh.util.Log.Debugf("File ID: %s", fileId)

	fi, err := fh.gateway.File.GetFile(usrInfo.AccountId, fileId)
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Error while attempting to query file: %v", err)
		return
	}
	if fi == nil {
		fh.util.HttpWriteCustomBadDataErrorf(w, ErrorBadDataMsg, ReasonFileDoesNotExist, "File ID %s does not exist", fileId)
		return
	}
	if fi.Type == file.FileTypeDir {
		// NOTE: this requires to create the files and zip it in the structure, recursively.
		// honestly it shouldnt be that hard to make but its not a priority.
		fh.util.HttpWriteBadDataError(w, "Directories are unsupported for downloading (file ID: %s)", fileId)
		return
	}

	filePath := filepath.Join(fh.gateway.Dir.Storage, fi.Path)
	f, err := os.Open(filePath)
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to open file: %v | Path: %s", err, fi.Path)
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
		fh.util.Log.Criticalf("Failed to write file to stream: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	fh.util.Log.Infof("Wrote %d bytes to stream", n)
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

	cinfo, ok := fh.uploadSessions[reqId]
	if !ok {
		cinfo = chunkInfo{
			TotalChunks:  totalChunks,
			ChunkIndexes: make(map[int]string),
			FileSize:     fileSize,
		}

		fh.util.Log.Infof("New chunk info created for %s, total chunks: %d", reqId, totalChunks)
		fh.uploadSessions[reqId] = cinfo
	}

	mutex := fh.getUploadMutex(reqId)

	mutex.Lock()
	defer mutex.Unlock()

	_, err = os.Stat(cinfo.Dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fh.util.Log.Criticalf("Failed to stat directory %s: %v", cinfo.Dir, err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	// if file does not exist or dir is empty
	if cinfo.Dir == "" || errors.Is(err, os.ErrNotExist) {
		fh.util.Log.Infof("No chunks directory found for request %s", reqId)

		reqChunkDir := filepath.Join(fh.gateway.Dir.Temp, CHUNKS_DIR_NAME, reqId)
		err = os.MkdirAll(reqChunkDir, 0o700)
		if err != nil {
			fh.util.Log.Criticalf("Failed to create directory: %v | Path: %s", err, reqChunkDir)
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}

		fh.util.Log.Infof("Directory set to %s", reqChunkDir)
		cinfo.Dir = reqChunkDir
		// setting directory to the map
		fh.uploadSessions[reqId] = cinfo
	}

	chunkName := fmt.Sprintf("chunk-i0%d-*", chunkIndex)
	chunkFile, err := os.CreateTemp(cinfo.Dir, chunkName)
	if err != nil {
		fh.util.Log.Criticalf("Failed to create temporary chunk file: %v | Path: %s", err, cinfo.Dir)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer chunkFile.Close()

	n, err := io.Copy(chunkFile, r.Body)
	if err != nil {
		fh.util.Log.Criticalf("Failed to write response body to file: %v", err)
		remErr := os.Remove(chunkFile.Name())
		if remErr != nil {
			fh.util.Log.Criticalf("Failed to remove chunk file: %v", err)
		}
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	cinfo.ChunkIndexes[chunkIndex] = chunkFile.Name()
	fh.util.Log.Infof("Wrote %d bytes to %s", n, chunkFile.Name())
}

// CompleteUploadFile is used to indicate that the uploading request has been completed.
// This requires the request ID to exist in the chunk map and all chunks must be accounted
// for.
//
// If the request ID does not exist, the total chunks doesn't match the written chunks,
// or missing/invalid headers, then the request will be rejected.
//
// Upon a successful upload, it will return a FileResponse response.
//
// This requires the auth middleware.
func (fh *FileHandler) CompleteUploadFile(w http.ResponseWriter, r *http.Request) {
	userContext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	err := CheckRequestHeaderNotEmpty(r, []string{HEADER_UPLOAD_REQUEST_ID})
	if err != nil {
		msg := utils.ToUpperFirstChar(err.Error())
		fh.util.HttpWriteCustomBadDataError(w, msg, ReasonInvalidHeader, msg)
		return
	}

	reqId := r.Header.Get(HEADER_UPLOAD_REQUEST_ID)
	// not checked if empty as it can be nil
	var parentId *string
	headerParentId := r.Header.Get(HEADER_UPLOAD_PARENT_ID)
	if headerParentId != "" {
		parentId = &headerParentId
	}

	cinfo, ok := fh.uploadSessions[reqId]
	if !ok {
		errMsg := fmt.Sprintf("Request ID %s does not exist", reqId)
		fh.util.HttpWriteCustomBadDataError(w, errMsg, ReasonInvalidHeader, errMsg)
		return
	}
	if cinfo.TotalChunks != len(cinfo.ChunkIndexes) {
		fh.util.Log.Warnf("Total chunks %d does not match written chunks %d", cinfo.TotalChunks, len(cinfo.ChunkIndexes))
		WriteErrorResponse(w, "Missing chunks", http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	accountDir, err := fh.mkAccountDir(userContext.AccountId)
	if err != nil {
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	fileId := uuid.NewString()
	fh.util.Log.Debugf("Generated file ID: %s", fileId)
	storedFilePath := filepath.Join(accountDir, fileId)
	fh.util.Log.Debugf("File path: %s", storedFilePath)

	sf, err := os.Create(storedFilePath)
	if err != nil {
		fh.util.Log.Criticalf("Failed to open file: %v | Path: %s", err, storedFilePath)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer sf.Close()

	for i := range cinfo.TotalChunks {
		// anon function due to defer file.Close()
		err, errMsg, code, reason := func() (error, string, int, ReasonCode) {
			chunkPath, ok := cinfo.ChunkIndexes[i]
			if !ok {
				fh.util.Log.Warnf("Missing chunk index %d", i)
				err := fmt.Errorf("Chunk index %d is missing (total %d)", i, cinfo.TotalChunks)
				return err, "Missing chunks", http.StatusBadRequest, ReasonBadRequestData
			}

			cf, err := os.Open(chunkPath)
			if err != nil {
				fh.util.Log.Criticalf("Failed to open file: %v | Path: %s", err, chunkPath)
				return err, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError
			}
			defer cf.Close()

			n, err := io.Copy(sf, cf)
			// abort the process, delete the main file
			if err != nil {
				fh.util.Log.Criticalf("Failed to write to stored file: %v | Path: %s | Chunk index: %d", err, sf.Name(), i)

				remErr := os.Remove(sf.Name())
				if remErr != nil {
					fh.util.Log.Criticalf("Failed to remove file: %v", err)
				} else {
					fh.util.Log.Infof("Removed file %s", sf.Name())
				}

				return err, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError
			}

			fh.util.Log.Infof("Wrote %d bytes to %s (chunk index %d)", n, sf.Name(), i)

			return nil, "", http.StatusOK, ""
		}()

		if err != nil {
			fh.util.Log.Critical("Failed to merge chunk file")
			WriteErrorResponse(w, errMsg, code, reason)
			return
		}
	}

	parent, base := filepath.Split(storedFilePath)
	// file path stored in the database is relative: <account_id>/<file_id>
	fileEntryPath := filepath.Join(filepath.Base(parent), base)

	fileEntry := file.File{
		OwnerID:   userContext.AccountId,
		Name:      cinfo.FileName,
		Extension: cinfo.FileExtension,
		// will never be a directory, files are flattened in the storage
		Type:       file.FileTypeFile,
		Size:       int64(cinfo.FileSize),
		FileID:     fileId,
		ModifiedOn: time.Now().UTC(),
		// TODO: parent id
		ParentID: parentId,
		Path:     fileEntryPath,
	}

	fh.util.Log.Debugf("File entry: %s", fileEntry.StringClean())

	err = fh.gateway.File.AddFile(fileEntry)
	if err != nil {
		removeErr := os.Remove(storedFilePath)
		if err != nil {
			fh.util.Log.Criticalf("Failed to remove file: %v | Path: %s", removeErr, storedFilePath)
		}
		fh.util.HttpWriteInternalError(w, "Failed to add file to database: %v", err)
		return
	}

	fileRes := fileEntry.ToFileResponse()

	WriteResponse(w, NewApiResponse(fileRes))
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
	fh.util.Log.Debugf("Request query: %v", parentID)

	userContext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	files, err := fh.gateway.File.GetFilesByAccountIdAndParentId(userContext.AccountId, parentID)
	if err == dbgateway.FileDoesNotExistErr {
		fh.util.Log.Infof("Given file ID %s does not exist: %v", parentID, err)
		WriteErrorResponse(w, "Invalid file ID", http.StatusBadRequest, ReasonBadRequestData)
		return
	}
	if err != nil {
		fh.util.Log.Criticalf("Failed to retrieve files with session ID and parent folder ID: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	convertedFiles := file.ToFileResponses(files...)
	res := NewApiResponse(convertedFiles)
	n, err := WriteResponse(w, res)
	if err != nil {
		fh.util.Log.Criticalf("Failed to write response to client: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	fh.util.Log.Debugf("Response bytes: %d", n)
}

// mkAccountDir creates the directory of the account ID. It will return the directory path upon
// creation or if it already exists.
func (fh *FileHandler) mkAccountDir(accountId string) (string, error) {
	path := filepath.Join(fh.gateway.Dir.Storage, accountId)

	err := os.MkdirAll(path, 0o700)
	if err != nil {
		fh.util.Log.Criticalf("Failed to create account storage: %v | Path: %s", err, path)
		return "", err
	}

	return path, nil
}

// addUploadMutex adds a new entry of the given upload ID into
// the mutex map and return the created mutex.
//
// If the upload ID already exists, then it will return the mutex.
func (fh *FileHandler) addUploadMutex(uploadId string) *sync.Mutex {
	mutex, ok := fh.uploadMutexes[uploadId]
	if !ok {
		newMutex := sync.Mutex{}
		fh.uploadMutexes[uploadId] = &newMutex
		mutex = &newMutex

		fh.util.Log.Infof("New mutex created for upload ID %s", uploadId)
	}

	return mutex
}

// getUploadMutex retrieves the mutex associated with the upload ID.
//
// If the entry does not exist, then it will add the entry and return a new mutex.
func (fh *FileHandler) getUploadMutex(uploadId string) *sync.Mutex {
	mutex, ok := fh.uploadMutexes[uploadId]
	if !ok {
		return fh.addUploadMutex(uploadId)
	}

	return mutex
}

// clearMutext clears the entry of a given upload ID from the
// mutex map.
//
// If the upload ID does not exist, then it will do nothing.
func (fh *FileHandler) clearMutex(uploadId string) {
	_, ok := fh.uploadMutexes[uploadId]
	if ok {
		delete(fh.uploadMutexes, uploadId)
		fh.util.Log.Infof("Cleared mutex for upload ID %s", uploadId)
	} else {
		fh.util.Log.Infof("No mutex entry found for upload ID %s", uploadId)
	}
}
