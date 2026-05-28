package tests

import (
	"bytes"
	"encoding/json"
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

// NewRequestBody creates a new request body.
// It marshals v and returns the Reader of v.
func NewRequestBody(v any) (io.Reader, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)

	return buf, nil
}
