package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var (
	errInvalidJSON  = errors.New("request body is not valid JSON")
	errBodyTooLarge = errors.New("request body exceeds the size limit")
)

type runResponse struct {
	ID           domain.RunID             `json:"id"`
	State        domain.RunState          `json:"state"`
	Profile      domain.PermissionProfile `json:"profile"`
	Task         string                   `json:"task"`
	RepoRoot     string                   `json:"repo_root"`
	CurrentStage string                   `json:"current_stage"`
	CreatedAt    any                      `json:"created_at"`
	UpdatedAt    any                      `json:"updated_at"`
}

func newRunResponse(run domain.Run) runResponse {
	return runResponse{
		ID: run.ID, State: run.State, Profile: run.Profile, Task: run.Task,
		RepoRoot: run.RepoRoot, CurrentStage: run.CurrentStage,
		CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt,
	}
}

func (r *Router) createRun(writer http.ResponseWriter, request *http.Request, requestID string) {
	var input app.CreateRunRequest
	if !r.decodeJSON(writer, request, requestID, &input) {
		return
	}
	run, err := r.application.CreateRun(request.Context(), input)
	if err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	r.writeJSON(writer, http.StatusCreated, newRunResponse(run))
}

func (r *Router) getRun(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	runID domain.RunID,
) {
	run, err := r.application.GetRun(request.Context(), runID)
	if err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	r.writeJSON(writer, http.StatusOK, newRunResponse(run))
}

func (r *Router) cancelRun(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	runID domain.RunID,
) {
	if !r.decodeEmptyObject(writer, request, requestID) {
		return
	}
	if err := r.application.CancelRun(request.Context(), runID); err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (r *Router) validateConfig(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
) {
	var input app.CreateRunRequest
	if !r.decodeJSON(writer, request, requestID, &input) {
		return
	}
	report := r.application.Preflight(request.Context(), input)
	r.writeJSON(writer, http.StatusOK, report)
}

func (r *Router) artifact(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	runID domain.RunID,
	artifactID string,
) {
	artifact, err := r.store.GetArtifact(request.Context(), runID, artifactID)
	if err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	r.writeJSON(writer, http.StatusOK, struct {
		ID        string       `json:"id"`
		RunID     domain.RunID `json:"run_id"`
		Kind      string       `json:"kind"`
		SHA256    string       `json:"sha256"`
		Content   []byte       `json:"content"`
		Truncated bool         `json:"truncated"`
	}{
		ID: artifact.ID, RunID: artifact.RunID, Kind: artifact.Kind,
		SHA256:    artifact.SHA256,
		Content:   []byte(r.redact(string(artifact.Content))),
		Truncated: artifact.Truncated,
	})
}

func (r *Router) decodeEmptyObject(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, r.maxBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input struct{}
	if err := decoder.Decode(&input); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		r.writeDecodeError(writer, err, requestID)
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errInvalidJSON
		}
		r.writeDecodeError(writer, err, requestID)
		return false
	}
	return true
}

func (r *Router) decodeJSON(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	target any,
) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, r.maxBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		r.writeDecodeError(writer, err, requestID)
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errInvalidJSON
		}
		r.writeDecodeError(writer, err, requestID)
		return false
	}
	return true
}

func (r *Router) writeDecodeError(writer http.ResponseWriter, err error, requestID string) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) || errors.Is(err, errBodyTooLarge) {
		r.writeError(
			writer, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE",
			"request body exceeds the size limit", requestID,
		)
		return
	}
	r.writeError(
		writer, http.StatusBadRequest, "INVALID_JSON",
		"request body is not valid JSON", requestID,
	)
}
