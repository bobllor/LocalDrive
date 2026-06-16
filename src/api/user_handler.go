package api

import (
	"encoding/json"
	"net/http"
	"strings"

	dbcon "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/cloud-project/src/utils"
	"github.com/bobllor/gologger"
)

const (
	UserPostRegisterRoute = "POST /api/register"
	UserPostLoginRoute    = "POST /api/login"
	UserPostLogoutRoute   = "POST /api/logout"
	UserGetUserRoute      = "/api/user"
)

// TODO: add string checker for empty/invalid characters (username/password)
// TODO: add test cases for Login and other things

func NewUserHandler(gw *dbcon.Gateway, logger *gologger.Logger) *UserHandler {
	uh := &UserHandler{
		Gateway: gw,
		deps:    &utils.Deps{Log: logger},
	}

	return uh
}

// UserHandler contains handlers used for handling user related
// logic.
type UserHandler struct {
	Gateway *dbcon.Gateway
	deps    *utils.Deps
}

// GetUserBySessionID retrieves the user information to the response. This uses
// the cookie in the headers.
//
// This requires a valid session ID in order to retrieve the user. The user
// will only include the necessary information that is required with the frontend.
//
// If the user cannot be found, then the output will be nil in the Response.
// An error Response will only be returned for internal errors.
func (uh *UserHandler) GetUserBySessionID(w http.ResponseWriter, r *http.Request) {
	uh.deps.Log.Debugf("Request cookies size: %d", len(r.Cookies()))

	id := GetSessionFromCookie(r)
	if id == "" {
		uh.deps.Log.Info("No cookie found with request")
		WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)
		return
	}

	ua, err := uh.Gateway.User.GetUserBySessionID(id)
	if err != nil {
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	if ua == nil {
		WriteResponse(w, NewApiResponse[any](nil))
	} else {
		res := NewApiResponse(ua)
		n, err := WriteResponse(w, res)
		if err != nil {
			uh.deps.Log.Warnf("failed to write response: %v", err)
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)

			return
		}

		uh.deps.Log.Info("Completed user retrieval by session ID request")
		logResponseBytes(uh.deps.Log, n)
	}
}

// PostLogin is the handler for handling the login and authentication.
// The username and password will be validated if the cookie is not found
// with a valid session ID.
//
// The middleware does not effect this, but the session ID is to prevent reauth.
//
// If the user is successfully authenticated, the output of the response
// will contain the status and the session ID will be written to the cookie.
func (uh *UserHandler) PostLogin(w http.ResponseWriter, r *http.Request) {
	uh.deps.Log.Infof("Login handler accessed (%v)", r.RemoteAddr)
	var user RequestUserLoginInfo

	c, err := r.Cookie(CookieSessionKey)
	if err != nil {
		uh.deps.Log.Infof("Cookie key %s not found", CookieSessionKey)
	} else {
		validSession, _, err := uh.Gateway.Session.ValidateSessionAndGetUser(c.Value)
		if err != nil {
			uh.deps.Log.Warnf("Got an error while validating session: %v", err)
		}

		if validSession {
			// NOTE: this might be a bad idea. look at this another time
			_, err := WriteResponse(w, NewApiResponse(validSession))
			if err != nil {
				WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			} else {
				return
			}
		}
	}

	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)
		return
	}

	// lowercasing and removing spaces for validation, the username is case sensitive
	user.Username = strings.TrimSpace(strings.ToLower(user.Username))
	validUser, ua, err := uh.Gateway.User.ValidateUser(user.Username, user.Password)
	if err != nil {
		uh.deps.Log.Criticalf("Error occurred during user validation: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	uh.deps.Log.Debugf("%s result: %v", user.Username, validUser)

	res := NewApiResponse(validUser)
	if validUser {
		// a new login will always create a new session ID
		ses, err := uh.Gateway.Session.UpsertSession(ua.AccountID)
		if err != nil {
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		}

		uh.deps.Log.Infof("Setting session data for user %s", ua.Username)
		SetCookieSession(w, ses.SessionID)
		_, err = WriteResponse(w, res)
		if err != nil {
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		}
	} else {
		uh.deps.Log.Info("User does not exist")
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)
	}

	uh.deps.Log.Info("Completed login request")
}

// PostLogout is the handler for invalidating the user.
// This requires the cookie for the session ID.
//
// The session ID will be deleted from the table and the cookie will be removed
// from the browser.
//
// A boolean value will be returned to the response output if successful.
func (uh *UserHandler) PostLogout(w http.ResponseWriter, r *http.Request) {
	id := GetSessionFromCookie(r)
	if id == "" {
		WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusBadRequest, ReasonUnauthorized)
		return
	}

	err := uh.Gateway.Session.DeleteSessionByID(id)
	if err != nil {
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	ExpireCookieSession(w)

	n, err := WriteResponse(w, NewApiResponse(true))
	if err != nil {
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	uh.deps.Log.Info("Completed logout request")
	logResponseBytes(uh.deps.Log, n)
}

// PostRegisterUser is the handler for registering users.
//
// If successful, the session ID will be written to the cookie. The output will
// only consist of a boolean value.
func (uh *UserHandler) PostRegisterUser(w http.ResponseWriter, r *http.Request) {
	var user RequestUserRegisterInfo

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		uh.deps.Log.Warnf("Failed to decode JSON: %v", err)
		WriteErrorResponse(w, ErrorBadDataMsg, http.StatusBadRequest, ReasonBadRequestData)

		return
	}

	// lowercasing and removing spaces for validation, the username is case sensitive
	user.Username = strings.TrimSpace(strings.ToLower(user.Username))
	acc, err := uh.Gateway.User.AddUser(user.Username, user.Password)
	if err != nil {
		if dbcon.IsDuplicateSqlError(err) {
			WriteErrorResponse(w, "Username already exists", http.StatusBadRequest, ReasonUserAlreadyExists)
		} else if dbcon.IsUsernameError(err) {
			// the error for username validation already contains the string for the frontend
			WriteErrorResponse(w, err.Error(), http.StatusBadRequest, ReasonBadUsername)
		} else {
			uh.deps.Log.Warnf("Failed to add user: %v", err)
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		}

		return
	}

	s, err := uh.Gateway.Session.UpsertSession(acc.AccountID)
	if err != nil {
		uh.deps.Log.Warnf("Failed to upsert session: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	SetCookieSession(w, s.SessionID)

	a := NewApiResponse(true)
	n, err := WriteResponse(w, a)
	if err != nil {
		uh.deps.Log.Warnf("Failed to write response: %v", err)
		WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
		return
	}

	uh.deps.Log.Info("Completed user registration request")
	logResponseBytes(uh.deps.Log, n)
}
