package dipod

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/moby/api/types"
)

// WriteError returns an error response to the client.
func WriteError(res http.ResponseWriter, err error) {
	JSONResponse(res, types.ErrorResponse{Message: ErrorMessage(err)})
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
