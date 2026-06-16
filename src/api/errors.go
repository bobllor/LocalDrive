package api

var (
	// ReasonInternalError is used for an internal server error. This should only
	// be used for CRITICAL and FATAL level errors.
	ReasonInternalError ReasonCode = "INTERNAL_ERROR"

	// ReasonBadRequestData is used if the response body is invalid or fails to meet
	// requirements of a method when its consumed. This is a generic bad data request.
	ReasonBadRequestData ReasonCode = "BAD_DATA"

	// ReasonDuplicateData is used for bad request data that resulted in a duplicate
	// SQL error.
	ReasonDuplicateData ReasonCode = "DUPLICATE_DATA"

	// ReasonFileDoesNotExist is used if the file does not have an entry in the database.
	ReasonFileDoesNotExist ReasonCode = "FILE_DOES_NOT_EXIST"

	// ReasonInvalidHeader is used for header values that are invalid.
	ReasonInvalidHeader ReasonCode = "INVALID_HEADER"

	// ReasonUserAlreadyExists is used for when the SQL database rejects the user
	// due to a duplicate entry.
	ReasonUserAlreadyExists ReasonCode = "USER_ALREADY_EXISTS"

	// ReasonBadUsername is used to indicate the given username failed validation.
	ReasonBadUsername ReasonCode = "BAD_USERNAME"

	// ReasonBadPassword is used to indicate the given password failed validation.
	ReasonBadPassword ReasonCode = "BAD_PASSWORD"

	// ReasonUnauthorized is used for unauthenticated requests.
	ReasonUnauthorized ReasonCode = "UNAUTHORIZED"
)

var (
	ErrorInternalErrorMsg = "An unexpected internal error has occurred"
	ErrorUnauthorizedMsg  = "Unauthorized access"
	ErrorBadDataMsg       = "Bad request data"
)
