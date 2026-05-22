package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bobllor/assert"
	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/tests"
)

func TestAuthMiddlewareApi(t *testing.T) {
	type testCases struct {
		Name       string
		SessionId  string
		IsErr      bool
		SkipCookie bool
	}

	gw, _ := dbgateway.NewTestGatewayDB(t)
	ap := NewApiHandler(gw, tests.NewTestLogger())
	pattern := "/api/example"

	mux := http.NewServeMux()
	f := func(w http.ResponseWriter, r *http.Request) {
		WriteResponse(w, NewApiResponse(true))
	}

	serv := httptest.NewServer(mux)
	defer serv.Close()

	cases := []testCases{
		{
			Name:      "Successful validation",
			SessionId: tests.DbRowInfo.SessionID,
			IsErr:     false,
		},
		{
			Name:  "Empty session ID",
			IsErr: true,
		},
		{
			Name:      "Invalid session ID",
			SessionId: "1234-1555",
			IsErr:     true,
		},
		{
			Name:       "No cookie",
			SessionId:  tests.DbRowInfo.SessionID,
			IsErr:      true,
			SkipCookie: true,
		},
	}
	mux.Handle(pattern, ap.CreateAuthMiddleware(f))

	client := serv.Client()

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			req, err := http.NewRequest("GET", serv.URL+pattern, bytes.NewBuffer([]byte{}))
			assert.Nil(t, err)

			if !c.SkipCookie {
				req.AddCookie(
					&http.Cookie{
						Name:  CookieSessionKey,
						Value: c.SessionId,
					},
				)
			}

			res, err := client.Do(req)
			assert.Nil(t, err)

			var apiRes ApiResponse
			err = json.NewDecoder(res.Body).Decode(&apiRes)
			assert.Nil(t, err)

			if !c.IsErr {
				assert.Equal(t, apiRes.Status, StatusSuccess)
				assert.Equal(t, apiRes.Output, true)
			} else {
				assert.Equal(t, apiRes.Status, StatusError)
				assert.NotNil(t, apiRes.Error)

				assert.Equal(t, apiRes.Error.Reason, ReasonUnauthorized)
			}
		})
	}
}

func TestAuthMiddlewareContext(t *testing.T) {
	gw, _ := dbgateway.NewTestGatewayDB(t)
	ap := NewApiHandler(gw, tests.NewTestLogger())
	urlPattern := "/api/test"

	f := func(w http.ResponseWriter, r *http.Request) {
		usr := r.Context().Value(CONTEXT_USER_SESSION_KEY)
		if usr == nil {
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}

		// NOTE: this is not a real response from a handler,
		// this is only used for testing to check the value exists from the context
		// handler responses are wrapped in a ResponseApi struct
		WriteResponse(w, usr)
	}

	mux := http.NewServeMux()
	mux.Handle(urlPattern, ap.CreateAuthMiddleware(f))

	serv := httptest.NewServer(mux)
	defer serv.Close()
	url := serv.URL

	client := serv.Client()

	req, err := http.NewRequest("GET", url+urlPattern, bytes.NewBuffer([]byte{}))
	assert.Nil(t, err)
	req.AddCookie(&http.Cookie{
		Name:  CookieSessionKey,
		Value: tests.DbRowInfo.SessionID,
	})

	res, err := client.Do(req)
	assert.Nil(t, err)
	defer res.Body.Close()

	var userSes dbgateway.UserSessionInfo
	err = json.NewDecoder(res.Body).Decode(&userSes)
	assert.Nil(t, err)

	assert.NotNil(t, userSes)
	assert.Equal(t, userSes.AccountId, tests.DbRowInfo.AccountID)
}
