package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/gologger"
	"github.com/google/uuid"
)

const (
	FileGetFileRootRoute            = "GET /api/storage"
	FileGetFileParentRoute          = "GET /api/storage/folder/{parentId}"
	FileGetFolderBreadcrumbsRoute   = "GET /api/folders/{folderId}/breadcrumbs"
	FilePostUploadFileRoute         = "POST /api/upload"
	FilePostUploadFileChunkRoute    = "POST /api/upload/{id}/{chunkIndex}"
	FilePostUploadFileCompleteRoute = "POST /api/upload/{id}/complete"
	FilePostDownloadFileRoute       = "GET /api/download/file/{fileId}"
	FilePostAddFolderRoute          = "POST /api/folders/add"
	FilePatchUpdateFileStatusRoute  = "PATCH /api/upload/{id}/fail"
	FilePatchRenameFileRoute        = "PATCH /api/file/rename"
	FilePatchRestoreFileRoute       = "PATCH /api/file/restore/{id}"
	FileDeleteFileDeleteionRoute    = "DELETE /api/file/delete/{id}"
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

	// uploadMut is the map of mutexes of a given upload ID for uploading files. This ensures that
	// chunks will not be modified or accessed while one is already active.
	uploadMutex *MutexMap

	// completeMutexes is the map of mutexes of a given upload ID for completion of the uploading files.
	// This prevents overwriting or accessing the same file during chunk merging.
	completeMutexes *MutexMap
}

type chunkInfo struct {
	// TotalChunks is the number of chunks that is expected with the upload.
	TotalChunks int

	// Dir is the directory where the chunks are stored. This is set during the file chunk
	// uploads.
	Dir string

	// AccountId is the account ID of the file owner.
	AccountId string

	FileInfo file.File

	// ChunkIndexes is a map of indexes used to track the chunks. The value
	// is the absolute path to the written chunk.
	ChunkIndexes map[int]string
}

// osCreate creates a file and returns an os.File for use.
// This is the same as os.Create.
var osCreate = os.Create

func NewFileHandler(gw *dbgateway.Gateway, logger *gologger.Logger) *FileHandler {
	return &FileHandler{
		gateway:         gw,
		util:            &Utility{Log: logger},
		uploadSessions:  make(map[string]chunkInfo),
		uploadMutex:     NewMutexMap(),
		completeMutexes: NewMutexMap(),
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

	fileId := r.PathValue("fileId")
	if fileId == "" {
		fh.util.HttpWriteBadDataError(w, "Path value %s is empty", "fileId")
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
	// recreation of the file name will include the period again.
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

// UploadGenerateId generates an upload session ID for use by the client in uploading files.
// The file metadata is also added to the database upon success.
//
// This requires the auth middleware.
func (fh *FileHandler) UploadGenerateId(w http.ResponseWriter, r *http.Request) {
	userContext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	uploadId := uuid.NewString()

	var meta RequestFileUploadInfo
	err := json.NewDecoder(r.Body).Decode(&meta)
	if err != nil {
		fh.util.HttpWriteBadDataError(w, "Failed to decode request body for upload ID generation: %v", err)
		return
	}
	defer r.Body.Close()

	badData := []string{}
	if strings.TrimSpace(meta.FileName) == "" {
		badData = append(badData, "file name cannot be empty")
	}
	if meta.TotalChunks == 0 {
		badData = append(badData, "total chunks cannot be 0")
	}
	if len(badData) > 0 {
		dataErrMsg := strings.Join(badData, ", ")
		errMsg := fmt.Sprintf("Invalid data: '%s'", dataErrMsg)
		fh.util.HttpWriteCustomBadDataErrorf(w, errMsg, ReasonBadRequestData, "Invalid request body metadata: %s", dataErrMsg)
		return
	}

	// will never be a directory, files are flattened in the storage
	// directories are added in a fh.PostAddFolder
	entry := file.NewFile(
		userContext.AccountId,
		meta.FileName,
		file.FileTypeFile,
		strings.TrimPrefix(meta.FileExtension, "."),
		int64(meta.FileSize),
		meta.FileParentId,
		file.UploadPending,
	)

	fh.util.Log.Debugf("Generated file ID: %s", entry.FileID)

	fh.util.Log.Debugf("File entry (clean): %s", entry.StringClean())

	err = fh.gateway.File.AddFile(entry)
	if dbgateway.IsDuplicateSqlError(err) {
		fh.util.HttpWriteCustomBadDataError(
			w,
			fmt.Sprintf("Duplicate file %s.%s", entry.Name, entry.Extension),
			ReasonDuplicateData,
			fmt.Sprintf("Duplicate file entry found for %s.%s", entry.Name, entry.Extension),
		)
		return
	}
	if errors.Is(err, dbgateway.NoFileArgsErr) {
		fh.util.HttpWriteBadDataError(w, "No files given: %v", err)
		return
	}
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to add file to database: %v", err)
		return
	}

	fh.uploadSessions[uploadId] = chunkInfo{
		AccountId:    userContext.AccountId,
		TotalChunks:  meta.TotalChunks,
		FileInfo:     entry,
		ChunkIndexes: make(map[int]string),
	}
	fh.util.Log.Infof("New chunk info created for %s, total chunks: %d", uploadId, meta.TotalChunks)

	WriteResponse(w, NewApiResponse(uploadId))
}

// UploadFileChunk uploads a file chunk to the backend. The uploaded file can be sent in multiple chunks,
// and the handler is expected to be called multiple times.
// The chunks will be created in the temporary directory given in Gateway.Dir.Temp.
//
// If an error occurs while the chunk is writing, it will set the upload status of the file as
// failed. A failed entry will only be considered if the writing failed
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
func (fh *FileHandler) UploadFileChunk(w http.ResponseWriter, r *http.Request) {
	_, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	uploadId := r.PathValue("id")
	cinfo, ok := fh.uploadSessions[uploadId]
	// typically this is either unauthorized or an invalid id is given
	if !ok {
		fh.util.HttpWriteCustomBadDataErrorf(w, "Invalid upload ID", ReasonBadRequestData, "Invalid upload session ID given: %s", uploadId)
		return
	}
	chunkValue := r.PathValue("chunkIndex")
	chunkIndex, err := strconv.Atoi(chunkValue)
	if err != nil || chunkValue == "" {
		errMsg := fmt.Sprintf("Chunk index path is not a number (got %s)", chunkValue)
		WriteErrorResponse(w, errMsg, http.StatusBadRequest, ReasonInvalidHeader)
		return
	}
	// chunkIndex is 0-indexed
	if chunkIndex+1 > cinfo.TotalChunks {
		fh.util.HttpWriteCustomBadDataErrorf(
			w,
			fmt.Sprintf(
				"Invalid chunk index. Given chunk index is greater than the expected total chunks (%d>%d)",
				chunkIndex+1, cinfo.TotalChunks,
			),
			ReasonBadRequestData,
			"Invalid chunk given: chunk index %d is greater than total chunks %d",
			chunkIndex,
			cinfo.TotalChunks,
		)
		return
	}
	if cinfo.TotalChunks == len(cinfo.ChunkIndexes) {
		fh.util.HttpWriteCustomBadDataErrorf(
			w,
			"All chunks have been uploaded. Call the complete endpoint to finalize the upload.",
			ReasonBadRequestData,
			"Written chunks is equal to total chunks for session %s (%d=%d)",
			uploadId, cinfo.TotalChunks, len(cinfo.ChunkIndexes),
		)
		return
	}
	mutex := fh.uploadMutex.Get(uploadId)

	mutex.Lock()
	defer mutex.Unlock()

	_, err = os.Stat(cinfo.Dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fh.util.Log.Criticalf("Failed to stat directory %s: %v", cinfo.Dir, err)

		fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)

		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	// if file does not exist or dir is empty
	if cinfo.Dir == "" || errors.Is(err, os.ErrNotExist) {
		fh.util.Log.Infof("No chunks directory found for request %s", uploadId)

		reqChunkDir := filepath.Join(fh.gateway.Dir.Temp, CHUNKS_DIR_NAME, uploadId)
		err = os.MkdirAll(reqChunkDir, 0o700)
		if err != nil {
			fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)

			fh.util.Log.Criticalf("Failed to create directory: %v | Path: %s", err, reqChunkDir)

			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}

		fh.util.Log.Infof("Directory set to %s", reqChunkDir)
		cinfo.Dir = reqChunkDir
		// setting directory to the map
		fh.uploadSessions[uploadId] = cinfo
	}

	chunkName := fmt.Sprintf("chunk-i0%d-*", chunkIndex)
	chunkFile, err := os.CreateTemp(cinfo.Dir, chunkName)
	if err != nil {
		fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)

		fh.util.Log.Criticalf("Failed to create temporary chunk file: %v | Path: %s", err, cinfo.Dir)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer chunkFile.Close()
	defer r.Body.Close()

	contentType := r.Header.Get("content-type")

	fh.util.Log.Debugf("Content length: %d | Content type: %s", r.ContentLength, contentType)

	n, err := io.Copy(chunkFile, r.Body)
	if err != nil {
		fh.util.Log.Criticalf("Failed to write response body to file: %v", err)
		remErr := os.Remove(chunkFile.Name())
		if remErr != nil {
			fh.util.Log.Criticalf("Failed to remove chunk file: %v", err)
		}

		fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	cinfo.ChunkIndexes[chunkIndex] = chunkFile.Name()
	fh.util.Log.Infof("Wrote %d bytes to %s", n, chunkFile.Name())

	// i dont think this should be considered a fail since the file
	// is now on the device. for now just leave it alone.
	wn, err := WriteResponse(w, NewApiResponse(true))
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to write response: %v", err)
		return
	}

	fh.util.Log.Infof("Wrote %d bytes to response for chunk upload", wn)
}

// UploadFileComplete is used to signal that the uploading files for a upload ID has been completed.
// This requires the upload ID to exist in the chunk map and all chunks must be accounted for.
//
// The file name is required in the headers and cannot be an empty string. The extension
// is expected to be included in the file name, and is extracted out for its value. If
// an extension does not exist, then it will be a generic file.
//
// If the upload ID does not exist, the total chunks doesn't match the written chunks,
// missing/invalid headers, or any general file writing errors, then the request will be rejected.
// The file will be removed from the storage.
//
// Upon a successful upload, it will return a FileResponse response. This is used in the front end
// to update the UI for the file.
//
// If there are any errors with disk operations, it will be considered a failed attempt and will
// update the file entry to 'failed'.
//
// This requires the auth middleware.
func (fh *FileHandler) UploadFileComplete(w http.ResponseWriter, r *http.Request) {
	_, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}
	uploadId := r.PathValue("id")
	if uploadId == "" {
		fh.util.HttpWriteCustomBadDataError(w, "Invalid upload ID", ReasonBadRequestData, "Empty upload ID given")
		return
	}

	cinfo, ok := fh.uploadSessions[uploadId]
	if !ok {
		errMsg := fmt.Sprintf("Request ID %s does not exist", uploadId)
		fh.util.HttpWriteCustomBadDataError(w, errMsg, ReasonInvalidHeader, errMsg)
		return
	}
	if cinfo.TotalChunks != len(cinfo.ChunkIndexes) {
		fh.util.Log.Warnf("Total chunks %d does not match written chunks %d", cinfo.TotalChunks, len(cinfo.ChunkIndexes))
		WriteErrorResponse(w, "Missing chunks", http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	accountDir, err := fh.mkAccountDir(cinfo.AccountId)
	if err != nil {
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)

		fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)

		return
	}
	storedFilePath := filepath.Join(accountDir, cinfo.FileInfo.FileID)
	fh.util.Log.Debugf("File path: %s", storedFilePath)

	sf, err := osCreate(storedFilePath)
	if err != nil {
		fh.util.Log.Criticalf("Failed to create file: %v | Path: %s", err, storedFilePath)

		fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)

		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer sf.Close()
	// used at the end to remove the chunks if the request is successful
	chunkPathsToRemove := []string{}

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

				fh.removeFile(sf.Name())

				return err, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError
			}

			fh.util.Log.Infof("Wrote %d bytes to %s (chunk index %d)", n, sf.Name(), i)
			chunkPathsToRemove = append(chunkPathsToRemove, chunkPath)

			return nil, "", http.StatusOK, ""
		}()

		if err != nil {
			fh.util.Log.Criticalf("Failed to merge chunk file: %v (chunk_index=%d)", err, i)

			fh.removeFile(sf.Name())
			fh.updateUploadStatusFailed(cinfo.AccountId, cinfo.FileInfo.FileID)

			WriteErrorResponse(w, errMsg, code, reason)
			return
		}
	}

	// status update is complete
	cinfo.FileInfo.UploadStatus = file.UploadCompleted
	upErr := fh.gateway.File.UpdateFile(cinfo.AccountId, cinfo.FileInfo.FileID, file.ColumnUploadStatus, cinfo.FileInfo.UploadStatus)
	if upErr != nil {
		// errors will not cancel, the file will still be shown on the front end thanks to the response.
		// background workers will resolve the issue periodically.
		fh.util.Log.Criticalf("Failed to update file %s: %v", cinfo.FileInfo.FileID, err)
	}
	fileRes := cinfo.FileInfo.ToFileResponse()

	WriteResponse(w, NewApiResponse(fileRes))

	fh.util.Log.Infof("Removing %s from cache", uploadId)
	fh.uploadMutex.Remove(uploadId)
	delete(fh.uploadSessions, uploadId)
	// TODO: this will probably need a lock, research it.
	fh.removeFiles(chunkPathsToRemove...)
}

// UploadFileStatusFailed is used to mark an upload session's File entry as
// failed in the database.
// It writes a boolean ResponseApi if successful.
//
// This requires auth and is a PATCH request.
func (fh *FileHandler) UploadFileStatusFailed(w http.ResponseWriter, r *http.Request) {
	_, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	sessionId := r.PathValue("id")
	if sessionId == "" {
		fh.util.HttpWriteCustomBadDataError(
			w,
			"No session ID found.",
			ReasonBadRequestData,
			"No session ID was given",
		)
		return
	}

	cinfo, ok := fh.uploadSessions[sessionId]
	if !ok {
		fh.util.HttpWriteCustomBadDataErrorf(
			w,
			"Invalid session ID given.",
			ReasonBadRequestData,
			"Invalid session ID %s",
			sessionId,
		)
		return
	}

	err := fh.gateway.File.UpdateUploadStatus(cinfo.AccountId, cinfo.FileInfo.FileID, file.UploadFailed)
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to update file ID %s to failed: %v", cinfo.FileInfo.FileID, err)
		return
	}

	n, err := WriteResponse(w, NewApiResponse(true))
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to write response: %v", err)
		return
	}

	fh.util.Log.Debugf("Wrote %d bytes to response", n)

	fh.uploadMutex.Remove(sessionId)
	delete(fh.uploadSessions, sessionId)
}

// PostAddFolder adds a folder to a user. It uses a response body containing
// the folder name and folder's parent ID. The body will contain the
// FileResponse struct for use on the front end immediately after the call ends.
//
// Folders are only metadata, no physical files are created. If the folder name is
// empty, it will default to "New folder".
//
// Auth middleware is required.
func (fh *FileHandler) PostAddFolder(w http.ResponseWriter, r *http.Request) {
	usr, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	var folderReqBody RequestAddFolderInfo
	err := json.NewDecoder(r.Body).Decode(&folderReqBody)
	if err != nil {
		fh.util.HttpWriteBadDataError(w, "Failed to parse request body: %v", err)
		return
	}
	defer r.Body.Close()

	fh.util.Log.Debugf("Request body: %v", folderReqBody)
	if folderReqBody.Name == "" {
		fh.util.Log.Info("Empty folder name, changing to 'New Folder'")
		folderReqBody.Name = "New folder"
	}

	// folders are not written to the disk
	folderFile := file.NewFile(
		usr.AccountId,
		folderReqBody.Name,
		file.FileTypeDir,
		"",
		0,
		folderReqBody.ParentId,
		file.UploadCompleted,
	)

	dbErr := fh.gateway.File.AddFile(folderFile)
	if dbErr != nil {
		fh.util.HttpWriteInternalError(w, "Failed to add new folder to database: %v", err)
		return
	}

	fh.util.Log.Infof("Created folder '%s' (parentId=%s,id=%s)", folderFile.Name, folderFile.ParentID, folderFile.FileID)

	n, err := WriteResponse(w, NewApiResponse(folderFile.ToFileResponse()))
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to write response: %v", err)
		return
	}

	fh.util.Log.Infof("Wrote %d bytes to response body", n)
}

// GetFiles retrieves all files associated with the account ID and the given
// parent folder ID. By default it will ignore deleted files.
//
// The method also supports retrieving deleted files only that are
// associated with an account ID by using including the query "type=trash".
//
// It returns a *ApiResponse[[]FileResponse].
//
// If a parent folder ID is given, it will retrieve those parent folder files.
// If parent folder is empty, then it will retrieve the files with no parent or the
// root children.
//
// This requires the auth middleware wrapper due to the context.
func (fh *FileHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("parentId")
	fh.util.Log.Debugf("Requested files parent ID: %v", parentID)

	userContext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	queryType := r.URL.Query().Get("type")
	var files []file.FileResponse
	var queryError error

	fh.util.Log.Debugf("Query type: %s", queryType)

	switch queryType {
	case "trash":
		afiles, err := fh.getDeletedFiles(userContext.AccountId)
		queryError = err

		files = afiles
	default:
		afiles, err := fh.getFiles(userContext.AccountId, parentID)
		queryError = err

		files = afiles
	}

	if queryError != nil {
		respErr := queryError.(*responseError)
		WriteErrorResponse(w, respErr.Message, respErr.Code, respErr.Reason)
		return
	}

	fh.util.Log.Debugf("%d files found (parentId=%s)", len(files), parentID)
	res := NewApiResponse(files)
	n, err := WriteResponse(w, res)
	if err != nil {
		fh.util.Log.Criticalf("Failed to write response to client: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	fh.util.Log.Debugf("Response bytes: %d", n)
}

// GetFolderBreadcrumbs retrieves all parents of a given folder ID for breadcrumb navigation.
//
// The return slice will always have the following:
//   - The first entry will be the top level folder in the root
//   - The last entry will be the given folder ID
//
// Other entries will exist depending on the nesting level.
// Invalid IDs or an empty slice is considered a bad request.
//
// This requires auth.
func (fh *FileHandler) GetFolderBreadcrumbs(w http.ResponseWriter, r *http.Request) {
	usercontext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}
	folderId := r.PathValue("folderId")
	if folderId == "" {
		fh.util.HttpWriteCustomBadDataError(w, "No folder ID given", ReasonBadRequestData, "No folder ID given")
		return
	}

	folders, err := fh.gateway.File.GetBreadcrumbs(usercontext.AccountId, folderId)
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to retrieve folders: %v", err)
		return
	}
	if len(folders) == 0 {
		fh.util.HttpWriteCustomBadDataErrorf(w, "File not found", ReasonBadRequestData, "No folders found for ID %s", folderId)
		return
	}

	n, err := WriteResponse(w, NewApiResponse(folders))
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to write resposne: %v", err)
	}

	fh.util.Log.Debugf("Wrote %d bytes to response with folders", n)
}

// RestoreFiles restores a deleted file.
func (fh *FileHandler) RestoreFiles(w http.ResponseWriter, r *http.Request) {

}

// RenameFile renames a file via a PATCH request. Upon a successful name change, it will return back
// the file response with the updated name in the response.
//
// It will recalculate the hash during the renaming attempt. If the rename failed the uniqueness check,
// then it will return a 400 bad request.
//
// Folders are exempt from the uniqueness check and can have duplicate names.
//
// This requires auth.
func (fh *FileHandler) RenameFile(w http.ResponseWriter, r *http.Request) {
	context, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}

	var body *RequestRenameFile
	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(&body)
	if errors.Is(err, io.EOF) {
		fh.util.HttpWriteCustomBadDataError(
			w,
			"No body was given with the request",
			ReasonBadRequestData,
			"Empty request body given",
		)
		return
	}
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to decode rename file body: %v", err)
		return
	}

	if strings.TrimSpace(body.NewFileName) == "" {
		fh.util.HttpWriteCustomBadDataError(
			w,
			"File name cannot be empty",
			ReasonBadRequestData,
			"Given file name is empty",
		)
		return
	}
	if uuid.Validate(body.FileId) != nil {
		fh.util.HttpWriteCustomBadDataErrorf(
			w,
			"File ID is invalid",
			ReasonBadRequestData,
			"Invalid file ID given: %s", body.FileId,
		)
		return
	}

	fi, err := fh.gateway.File.RenameFile(context.AccountId, body.FileId, body.NewFileName)
	if err != nil {
		if dbgateway.IsDuplicateSqlError(err) {
			fh.util.HttpWriteCustomBadDataErrorf(
				w,
				fmt.Sprintf("File name %s already exists", body.NewFileName),
				ReasonDuplicateData,
				"Failed to rename file, new file name %s already exists (id=%s)", body.NewFileName, body.FileId,
			)
		} else {
			fh.util.HttpWriteInternalError(w, "Failed to rename file: %v | file_id=%s,file_name=%s", err, body.FileId, body.NewFileName)
		}
		return
	}

	n, err := WriteResponse(w, NewApiResponse(fi.ToFileResponse()))
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to write rename file response: %v", err)
		return
	}

	fh.util.Log.Debugf("Wrote %d bytes to response for rename file", n)
}

// DeleteFile soft deletes a file ID by setting the DeleteOn value to
// the time of the request was received.
//
// It will return a boolean response indicating its status. If all rows
// are successfully updated, then it will return a success.
func (fh *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	usrContext, ok := GetRequestContext[*dbgateway.UserSessionInfo](r, CONTEXT_USER_SESSION_KEY)
	if !ok {
		fh.util.HttpWriteUnauthorizedError(w, r)
		return
	}
	defer r.Body.Close()

	fileId := r.PathValue("id")
	if strings.TrimSpace(fileId) == "" {
		fh.util.HttpWriteCustomBadDataError(w, "Missing file ID", ReasonBadRequestData, "Given File ID for deletion is empty")
		return
	}

	deletedRows, err := fh.gateway.File.DeleteFiles(usrContext.AccountId, fileId)
	if err != nil {
		fh.util.HttpWriteInternalError(w, "An error occurred while deleting file (%s): %v", fileId, err)
		return
	}

	res := NewApiResponse(deletedRows == 1)
	// output will either be 1 or 0, if it is 0 then this has failed.
	if res.Output == false {
		resError := &Error{
			Code:    http.StatusNotFound,
			Reason:  ReasonNotFound,
			Message: "Failed to delete file, the file does not exist",
		}

		res.Status = StatusSuccess
		res.Error = resError
	}

	resBytes, err := WriteResponse(w, res)
	if err != nil {
		fh.util.HttpWriteInternalError(w, "Failed to write response: %v", err)
		return
	}

	if res.Output {
		fh.util.Log.Infof("Marked file for deletion (%s)", fileId)
	}

	fh.util.Log.Debugf("Wrote %d bytes to response", resBytes)
}

// mkAccountDir creates the directory of the account ID in the storage folder.
//
// It will return the directory path upon creation or if it already exists.
func (fh *FileHandler) mkAccountDir(accountId string) (string, error) {
	path := filepath.Join(fh.gateway.Dir.Storage, accountId)

	err := os.MkdirAll(path, 0o700)
	if err != nil {
		fh.util.Log.Criticalf("Failed to create account storage: %v | Path: %s", err, path)
		return "", err
	}

	return path, nil
}

// updateUploadStatusFailed updates the upload status of the file entry of the
// file ID owned by the account ID to 'failed'.
//
// This is used to ensure that the entry is 'failed' to allow duplicate entries
// for retries.
func (fh *FileHandler) updateUploadStatusFailed(accountId, fileId string) error {
	err := fh.gateway.File.UpdateUploadStatus(accountId, fileId, file.UploadFailed)
	if err != nil {
		fh.util.Log.Criticalf("Failed to update upload status (file_id=%s): %v", fileId, err)

		return err
	}

	return nil
}

// removeFile removes the file path and logs the result. Errors
// that occur are not returned but will be logged.
func (fh *FileHandler) removeFile(path string) {
	err := os.Remove(path)
	if err != nil {
		fh.util.Log.Criticalf("Failed to remove file: %v | Path: %s", err, path)
	} else {
		fh.util.Log.Infof("Removed file %s", path)
	}
}

// removeFiles removes the file paths and logs the result. Errors
// that occur are not returned but will be logged.
func (fh *FileHandler) removeFiles(paths ...string) {
	for _, path := range paths {
		err := os.Remove(path)
		if err != nil {
			fh.util.Log.Criticalf("Failed to remove file: %v | Path: %s", err, path)
		} else {
			fh.util.Log.Infof("Removed file %s", path)
		}
	}
}

// getFiles is a helper method to retrieve the files. It will return the file response
// and an error if one occurs.
//
// The error is a type responseError, which can be converted for additional information.
func (fh *FileHandler) getFiles(accountId, parentId string) ([]file.FileResponse, error) {
	files, err := fh.gateway.File.GetFilesByAccountIdAndParentId(accountId, parentId)
	if err == dbgateway.FileDoesNotExistErr {
		fh.util.Log.Infof("Given file ID %s does not exist: %v", parentId, err)

		return nil, &responseError{
			Message: "Invalid file ID",
			Code:    http.StatusBadRequest,
			Reason:  ReasonBadRequestData,
		}
	}
	if err != nil {
		fh.util.Log.Criticalf("Failed to retrieve files with session ID and parent folder ID: %v", err)

		return nil, &responseError{
			Message: ErrorInternalErrorMsg,
			Code:    http.StatusInternalServerError,
			Reason:  ReasonInternalError,
		}
	}

	return file.ToFileResponses(files...), nil
}

// getDeletedFiles is a helper method that retrieves the deleted files. It will return the
// FileResponse and an error.
//
// The error is a type responseError, which can be converted for additional information.
func (fh *FileHandler) getDeletedFiles(accountId string) ([]file.FileResponse, error) {
	files, err := fh.gateway.File.GetDeletedFiles(accountId)
	if err != nil {
		fh.util.Log.Criticalf("An error occurred while retrieving deleted files: %v", err)

		return nil, &responseError{
			Message: ErrorInternalErrorMsg,
			Code:    http.StatusInternalServerError,
			Reason:  ReasonInternalError,
		}
	}

	return files, nil
}
