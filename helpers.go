package dipod

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/moby/api/types"
)

// ErrNotImplemented is returned when functionality requested was not
// implemented yet.
var ErrNotImplemented = errors.New("dipod: not implemented")

// WriteError returns an error response to the client.
func WriteError(res http.ResponseWriter, statusCode int, err error) {
	res.WriteHeader(statusCode)
	err = json.NewEncoder(res).Encode(
		types.ErrorResponse{Message: ErrorMessage(err)},
	)
	if err != nil {
		res.Write([]byte(err.Error()))
	}
}

// StreamError sends an error response to the client. Make sure to flush after.
func StreamError(res http.ResponseWriter, err error) {
	msg := jsonmessage.JSONMessage{
		Error: &jsonmessage.JSONError{
			Code: 0xDEAD,
		},
	}
	msg.Error.Message = ErrorMessage(err)
	JSONResponse(res, msg)
}

// JSONResponse returns a JSON-encoded response to the client.
func JSONResponse(res http.ResponseWriter, i interface{}) {
	err := json.NewEncoder(res).Encode(i)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
	}
}
