package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/bobllor/cloud-project/src/utils"
	"github.com/bobllor/gologger"
)

const (
	CookieSessionKey = "lcsSessionID"
)

type HandlerMap map[string]func(http.ResponseWriter, *http.Request)

type Handler interface{}

type Utility struct {
	Log *gologger.Logger
}

// HttpWriteUnauthorizedError writes a generic unauthorized error ot the ResponseWriter
// and logs a given message at the INFO level.
//
// It uses the context given in the http.Request to extract the request ID of the request.
// If the request ID does not exist, then it will log normally.
func (u *Utility) HttpWriteUnauthorizedError(w http.ResponseWriter, r *http.Request) {
	requestContext, _ := GetRequestContext[string](r, CONTEXT_REQUEST_ID_KEY)
	u.Log.Infof("Unauthorized access | Request ID: %s", requestContext)
	WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)
}

// HttpWriteInternalError writes a generic internal error to the ResponseWriter and logs
// a given message at the CRITICAL level.
//
// If an empty string is given for the format, then it will not log.
func (u *Utility) HttpWriteInternalError(w http.ResponseWriter, format string, v ...any) {
	if format != "" {
		u.Log.Criticalf(format, v...)
	}
	WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
}

// HttpWriteBadDataError writes a generic bad data error to the ResponseWriter and logs
// a given message at the WARN level.
//
// If an empty string is given for the format, then it will not log.
func (u *Utility) HttpWriteBadDataError(w http.ResponseWriter, format string, v ...any) {
	if format != "" {
		u.Log.Warnf(format, v...)
	}
	WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)
}

// HttpWriteCustomBadDataError writes a bad data error to the ResponseWriter with modications
// on the client error message and the reason code.
//
// It logs at the WARN level. If an empty string is given for the log message, then it will not log.
func (u *Utility) HttpWriteCustomBadDataError(w http.ResponseWriter, errMsg string, reason ReasonCode, logMsg string) {
	if logMsg != "" {
		u.Log.Warn(logMsg)
	}
	errMsg = utils.ToUpperFirstChar(errMsg)
	WriteErrorResponse(w, errMsg, http.StatusBadRequest, reason)
}

// HttpWriteCustomBadDataErrorf writes a bad data error to the ResponseWriter with modications
// on the client error message and the reason code.
//
// It logs at the WARN level.
func (u *Utility) HttpWriteCustomBadDataErrorf(w http.ResponseWriter, errMsg string, reason ReasonCode, format string, v ...any) {
	u.Log.Warnf(format, v...)
	errMsg = utils.ToUpperFirstChar(errMsg)
	WriteErrorResponse(w, errMsg, http.StatusBadRequest, reason)
}

// WriteErrorResponse is a helper function used to write an error to
// the ResponseWriter.
func WriteErrorResponse(w http.ResponseWriter, msg string, statusCode int, reason ReasonCode) {
	errRes := NewApiResponseError(statusCode, msg, reason)

	b, err := json.Marshal(errRes)
	// if err is not nil, then default to a basic value.
	// TODO: maybe find a fix for this. remove this line later
	if err != nil {
		http.Error(w, err.Error(), statusCode)
	} else {
		http.Error(w, string(b), statusCode)
	}

}

// WriteResponse writes a response to the ResponseWriter. If an error occurs
// while writing the response, it will return the error and the response will not
// be written.
//
// The bytes written will be returned.
func WriteResponse(w http.ResponseWriter, v any) (int, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}

	i, err := w.Write(b)
	if err != nil {
		return 0, err
	}

	return i, nil
}

// WriteHeaders writes the headers for CORS.
func WriteHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Vary", "Origin")
}

// GetSessionFromCookie retrieves the session ID from the request headers.
// If the cookie does not exist, then it will return an empty string.
func GetSessionFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(CookieSessionKey)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// SetCookieSession sets the cookie for the session.
// If the session already exists in the cookie, then it will overwrite the
// cookie's value.
func SetCookieSession(w http.ResponseWriter, id string) {
	c := http.Cookie{
		Name:     CookieSessionKey,
		Value:    id,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   true,
	}

	http.SetCookie(w, &c)
}

// ExpireCookieSession sets MaxAge=0 for the cookie session key to the
// ResponseWriter for the request.
func ExpireCookieSession(w http.ResponseWriter) {
	c := http.Cookie{
		Name:     CookieSessionKey,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	}

	http.SetCookie(w, &c)
}

// GetRequestContext retrieves the context value of type T from the http.Request.
//
// If the value is not of type T, it will return the zero value of T and false.
func GetRequestContext[T any](r *http.Request, contextKey ContextKey) (T, bool) {
	var z T

	v, ok := r.Context().Value(contextKey).(T)
	if !ok {
		return z, false
	}

	return v, true
}

// CheckEmptyRequestHeadersNotEmpty checks the request headers keys if it has
// a value and is not empty.
//
// It will return an error if a key is empty. The error will contain the missing
// key value.
func CheckRequestHeaderNotEmpty(r *http.Request, keys []string) error {
	missing := []string{}
	for _, k := range keys {
		val := strings.TrimSpace(r.Header.Get(k))
		if val == "" {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		joinedMissingKeys := strings.Join(missing, ",")
		msg := fmt.Sprintf("missing headers: %s", joinedMissingKeys)

		return errors.New(msg)
	}

	return nil
}

// logResponseBytes logs the bytes written to the response.
func logResponseBytes(log *gologger.Logger, n int) {
	log.Infof("Wrote %d bytes to response", n)
}
