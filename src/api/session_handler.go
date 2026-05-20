package api

import (
	"net/http"

	dbgateway "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/bobllor/gologger"
)

const (
	SessionGetValidateSessionRoute = "GET /api/session"
)

type SessionHandler struct {
	gateway *dbgateway.Gateway
	deps    *utils.Deps
}

func NewSessionHandler(gw *dbgateway.Gateway, logger *gologger.Logger) *SessionHandler {
	return &SessionHandler{
		gateway: gw,
		deps:    utils.NewDeps(logger),
	}
}

// ValidateSession is a GET request to validate the sessions based on the cookies.
//
// This is not used with middleware, and is only intended for public facing APIs.
func (sh *SessionHandler) GetValidateSession(w http.ResponseWriter, r *http.Request) {
	sesID := GetSessionFromCookie(r)

	status, _, err := sh.gateway.Session.ValidateSessionAndGetUser(sesID)
	if err != nil {
		sh.deps.Log.Criticalf("Error with validating session: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	res := NewApiResponse(true)
	if !status {
		res.Output = false
	}

	WriteResponse(w, res)
}
