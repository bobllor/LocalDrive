package api

import (
	"context"
	"net/http"
	"time"

	dbcon "github.com/bobllor/cloud-project/src/db_gateway"
	"github.com/bobllor/gologger"
	"github.com/google/uuid"
)

const (
	ContentJson = "application/json"
)

type ContextKey string

const (
	CONTEXT_USER_SESSION_KEY ContextKey = "userSession"
)

type ApiHandler struct {
	UserHandler    *UserHandler
	FileHandler    *FileHandler
	SessionHandler *SessionHandler
	gateway        *dbcon.Gateway
	log            *gologger.Logger
}

// NewApiHandler creates a new Api struct.
func NewApiHandler(gw *dbcon.Gateway, logger *gologger.Logger) *ApiHandler {
	api := &ApiHandler{
		UserHandler:    NewUserHandler(gw, logger),
		FileHandler:    NewFileHandler(gw, logger),
		SessionHandler: NewSessionHandler(gw, logger),
		gateway:        gw,
		log:            logger,
	}

	return api
}

// RequestMiddleware wraps a function with a middleware used to log and write headers by default.
//
// This does not handle auth, use ah.CreateAuthMiddleware for auth based middleware.
func (ah *ApiHandler) CreateRequestMiddleware(f func(http.ResponseWriter, *http.Request)) http.Handler {
	next := http.HandlerFunc(f)

	wrapper := func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}

	return ah.middlewareHandler(wrapper)
}

// CreateAuthHandler creates a new handler from a given handler function, wrapped in an
// authentication-based closure.
//
// Headers are automatically written if wrapped with this method.
//
// The UserSessionInfo will be attached in the context for use in the request.
func (ah *ApiHandler) CreateAuthMiddleware(f func(http.ResponseWriter, *http.Request)) http.Handler {
	next := http.HandlerFunc(f)

	wrapper := func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(CookieSessionKey)
		if err != nil {
			ah.log.Infof("Unauthorized access, no cookie found for %v", r.RemoteAddr)
			WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)

			return
		}

		validSession, ses, err := ah.gateway.Session.ValidateSessionAndGetUser(sessionCookie.Value)
		if err != nil {
			ah.log.Criticalf("Validating session failed: %v", err)
			WriteErrorResponse(w, ErrorInternalErrorMsg, http.StatusInternalServerError, ReasonInternalError)
			return
		}

		if !validSession {
			ah.log.Infof("Invalid session ID, unauthroized access from %v", r.RemoteAddr)
			WriteErrorResponse(w, ErrorUnauthorizedMsg, http.StatusUnauthorized, ReasonUnauthorized)

			return
		}

		r = r.WithContext(context.WithValue(r.Context(), CONTEXT_USER_SESSION_KEY, ses))

		// refreshes the cookie
		SetCookieSession(w, sessionCookie.Value)
		next.ServeHTTP(w, r)
	}

	return ah.middlewareHandler(wrapper)
}

// middlewareHandler is a generic handler used to create a new Handler wrapped in middleware.
//
// The given function is ran between a logging related tasks. The headers are
// automatically written within this method.
func (ah *ApiHandler) middlewareHandler(f func(http.ResponseWriter, *http.Request)) http.Handler {
	// expected to be wrapped function from the other method
	next := http.HandlerFunc(f)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestID := uuid.New().String()

		ah.log.Infof("Starting new request | id=%s,method=%s", requestID, r.Method)
		ah.log.Infof("%s: accessed on agent %s", r.RemoteAddr, r.UserAgent())

		WriteHeaders(w, r)

		next.ServeHTTP(w, r)

		finalTime := time.Since(startTime)
		ah.log.Infof(
			"Completed request | id=%s,time=%v seconds",
			requestID,
			finalTime.Seconds(),
		)
	})
}
