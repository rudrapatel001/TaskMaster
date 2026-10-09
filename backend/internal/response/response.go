package response

import (
	"encoding/json"
	"net/http"
)

// HeaderRequestID is the header that carries each request's unique ID.
// The request-ID middleware sets it on every response, and Error reads it back
// so error bodies can include the ID without every handler passing it around.
const HeaderRequestID = "X-Request-ID"

// Machine-readable error codes. Clients should branch on these, not on messages.
const (
	CodeBadRequest       = "bad_request"
	CodeValidation       = "validation_failed"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeNotFound         = "not_found"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeConflict         = "conflict"
	CodeRateLimited      = "rate_limited"
	CodeInternal         = "internal_error"
)

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type dataEnvelope struct {
	Data any `json:"data"`
}

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	// Encode first: if encoding fails, nothing has been sent yet and we can
	// still reply with a clean 500 instead of a half-written body.
	body, err := json.Marshal(v)
	if err != nil {
		Error(w, http.StatusInternalServerError, CodeInternal, "something went wrong")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// Data writes a successful response in the standard shape: {"data": ...}.
func Data(w http.ResponseWriter, status int, v any) {
	JSON(w, status, dataEnvelope{Data: v})
}

// Error writes a failure response in the standard shape:
// {"error": {"code": "...", "message": "...", "request_id": "..."}}.
func Error(w http.ResponseWriter, status int, code, message string) {
	body, _ := json.Marshal(errorEnvelope{Error: errorBody{
		Code:      code,
		Message:   message,
		RequestID: w.Header().Get(HeaderRequestID),
	}})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
