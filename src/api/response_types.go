package api

const (
	StatusSuccess StatusType = "success"
	StatusError   StatusType = "error"
)

// StatusType is the status of the response.
type StatusType string

// ReasonCode is a special code used to indicate what error type had occurred
// on the backend.
type ReasonCode string

type Error struct {
	// Code is the status code of the error.
	Code int `json:"statusCode"`

	// Reason is a special code used to indicate what error type had occurred
	// on the backend.
	Reason ReasonCode `json:"reason"`

	// Message is the string explaining why the error had occurred.
	Message string `json:"message"`
}

// ApiResponse is the response to the client from the server.
// This is the standardized response for the backend system for all handlers.
//
// The ApiResponse takes a type T which represents the Output of the response.
type ApiResponse[T any] struct {
	// Status is the status of the response.
	Status StatusType `json:"status"`

	// Output is the output data to the response. It can be omitted.
	Output T `json:"output,omitempty"`

	// Error is an Error type that occurred due to an issue with the request or backend.
	// It can be omitted.
	Error *Error `json:"error,omitempty"`
}

// NewApiResponse creates a new success API response for successful requests.
func NewApiResponse[T any](output T) *ApiResponse[T] {
	res := &ApiResponse[T]{
		Status: StatusSuccess,
		Output: output,
	}

	return res
}

// NewApiResponseError creates a new error API response.
//
// statusCode is the type of status code int.
//
// errMessage is the message string to send back to the client.
func NewApiResponseError(statusCode int, errMessage string, reason ReasonCode) *ApiResponse[any] {
	res := &ApiResponse[any]{
		Status: StatusError,
		Error: &Error{
			Code:    statusCode,
			Reason:  reason,
			Message: errMessage,
		},
	}

	return res
}
