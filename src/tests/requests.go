package tests

import (
	"bytes"
	"io"
	"net/http"
)

// NewRequest creates a new http.Request. This allows reader to be nil, and if it is
// nil then it will create an empty io.Reader for usage in the request.
func NewRequest(method string, url string, reader io.Reader) (*http.Request, error) {
	var useReader io.Reader
	if reader != nil {
		useReader = reader
	} else {
		useReader = bytes.NewBuffer([]byte{})
	}

	return http.NewRequest(method, url, useReader)
}

// GetCookie returns a test cookie containing the session ID
// of the default test DB data.
func GetCookie(sessionCookieName string) *http.Cookie {
	return &http.Cookie{
		Name:  sessionCookieName,
		Value: DbRowInfo.SessionID,
	}
}
