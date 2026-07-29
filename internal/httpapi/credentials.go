package httpapi

import (
	"errors"
	"net/http"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
)

func (r *Router) credential(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	provider string,
	host string,
) {
	ref := credentials.Ref{Provider: provider, Host: host}
	switch request.Method {
	case http.MethodGet:
		status, err := r.credentials.Status(request.Context(), ref)
		if err != nil {
			r.mapError(writer, err, requestID)
			return
		}
		r.writeJSON(writer, http.StatusOK, status)
	case http.MethodPut:
		var input struct {
			Secret string `json:"secret"`
		}
		if !r.decodeJSON(writer, request, requestID, &input) {
			return
		}
		r.registerSecret(input.Secret)
		secret := []byte(input.Secret)
		status, statusErr := r.credentials.Status(request.Context(), ref)
		var err error
		if statusErr == nil && status.Configured {
			err = r.credentials.Update(request.Context(), ref, secret)
		} else if statusErr == nil || errors.Is(statusErr, credentials.ErrNotFound) {
			err = r.credentials.Add(request.Context(), ref, secret)
		} else {
			err = statusErr
		}
		clearBytes(secret)
		input.Secret = ""
		if err != nil {
			r.mapError(writer, err, requestID)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if err := r.credentials.Clear(request.Context(), ref); err != nil {
			r.mapError(writer, err, requestID)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	default:
		r.writeMethodNotAllowed(writer, requestID)
	}
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
