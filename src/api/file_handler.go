package api

import (
	"fmt"
	"io"
	"net/http"
	"os"

	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/file"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/bobllor/gologger"
)

const PARENT_ID_KEY = "parentID"
const FILE_ID_DOWNLOAD_KEY = "fileId"

var FileGetFileParentRoute = fmt.Sprintf("GET /api/storage/folder/{%s}", PARENT_ID_KEY)
var FileDownloadFileRoute = fmt.Sprintf("POST /api/download/file/{%s}", FILE_ID_DOWNLOAD_KEY)

const (
	FileGetFileRootRoute = "GET /api/storage"
)

type FileHandler struct {
	gateway *dbgateway.Gateway
	deps    *utils.Deps
}

func NewFileHandler(gw *dbgateway.Gateway, logger *gologger.Logger) *FileHandler {
	return &FileHandler{
		gateway: gw,
		deps:    utils.NewDeps(logger),
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
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	f, err := os.Open(fi.Path)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to open file: %v | Path: %s", err, fi.Path)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}
	defer f.Close()

	n, err := io.Copy(w, f)
	if err != nil {
		fh.deps.Log.Criticalf("Failed to write file to stream: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	fh.deps.Log.Infof("Wrote %d bytes to stream", n)

	res := NewApiResponse(true)
	WriteResponse(w, res)
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
		requestContext, ok := GetRequestContext[string](r, CONTEXT_REQUEST_ID_KEY)
		if !ok {
			fh.deps.Log.Warn("Request ID is missing from middleware context")
			requestContext = ""
		}
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
