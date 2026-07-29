package httpapi

import (
	"net/http"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func (r *Router) approval(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	runID domain.RunID,
	digest string,
	decision string,
) {
	if digest == "" {
		r.writeError(writer, http.StatusBadRequest, "INVALID_APPROVAL", "approval digest is required", requestID)
		return
	}
	if decision == "approve" {
		if !r.decodeEmptyObject(writer, request, requestID) {
			return
		}
		if err := r.application.Approve(request.Context(), runID, digest); err != nil {
			r.mapError(writer, err, requestID)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	var input struct {
		Terminate bool `json:"terminate"`
	}
	if !r.decodeJSON(writer, request, requestID, &input) {
		return
	}
	if err := r.application.Reject(request.Context(), runID, digest, input.Terminate); err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
