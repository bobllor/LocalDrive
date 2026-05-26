package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bobllor/gologger"
)

const (
	CookieSessionKey = "lcsSessionID"
)

type HandlerMap map[string]func(http.ResponseWriter, *http.Request)

type Handler interface{}

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
	for _, k := range keys {
		val := strings.TrimSpace(r.Header.Get(k))
		if val == "" {
			return fmt.Errorf("missing header key %s from request", k)
		}
	}

	return nil
}

// logResponseBytes logs the bytes written to the response.
func logResponseBytes(log *gologger.Logger, n int) {
	log.Infof("Wrote %d bytes to response", n)
}
