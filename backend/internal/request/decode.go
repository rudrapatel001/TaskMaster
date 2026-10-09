package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/rudrapatel001/taskmaster/backend/internal/response"
)

// maxBodyBytes is the largest request body we will read (1 MiB).
const maxBodyBytes = 1 << 20

// DecodeJSON reads the JSON request body into dst, which must be a pointer.
// If anything is wrong with the body it writes the error response itself and
// returns false, so a handler only needs:
//
//	if !request.DecodeJSON(w, r, &input) { return }
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.Error(w, http.StatusUnsupportedMediaType,
			response.CodeUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeDecodeError(w, err)
		return false
	}

	// Decoding a second time must hit the end of the body. Anything else means
	// there was extra data after the first JSON value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.Error(w, http.StatusBadRequest,
			response.CodeBadRequest, "body must contain a single JSON object")
		return false
	}

	return true
}

// writeDecodeError turns Go's technical JSON errors into messages that are
// safe and useful to show to API clients.
func writeDecodeError(w http.ResponseWriter, err error) {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var tooLargeErr *http.MaxBytesError

	switch {
	case errors.As(err, &tooLargeErr):
		response.Error(w, http.StatusRequestEntityTooLarge, response.CodePayloadTooLarge,
			fmt.Sprintf("body must not be larger than %d bytes", tooLargeErr.Limit))

	case errors.Is(err, io.EOF):
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "body must not be empty")

	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "body contains malformed JSON")

	case errors.As(err, &typeErr):
		msg := "body contains a value of the wrong type"
		if typeErr.Field != "" {
			msg = fmt.Sprintf("field %q has the wrong type", typeErr.Field)
		}
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, msg)

	case strings.HasPrefix(err.Error(), "json: unknown field "):
		name := strings.TrimPrefix(err.Error(), "json: unknown field ")
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest,
			fmt.Sprintf("body contains unknown field %s", name))

	default:
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "body could not be read")
	}
}
