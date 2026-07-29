package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/app"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

var (
	errInvalidJSON       = errors.New("request body is not valid JSON")
	errBodyTooLarge      = errors.New("request body exceeds the size limit")
	errInvalidPagination = errors.New("pagination is invalid")
)

const (
	defaultRunPageLimit = 50
	maxRunPageLimit     = 100
	maxRunPageOffset    = 1_000_000
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

type runPageResponse struct {
	Runs []runResponse      `json:"runs"`
	Page paginationResponse `json:"page"`
}

type paginationResponse struct {
	Offset   int  `json:"offset"`
	Limit    int  `json:"limit"`
	Returned int  `json:"returned"`
	HasMore  bool `json:"has_more"`
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

func (r *Router) listRuns(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
) {
	offset, limit, err := parseRunPagination(request.URL.Query())
	if err != nil {
		r.writeError(
			writer, http.StatusBadRequest, "INVALID_PAGINATION",
			"run pagination is invalid", requestID,
		)
		return
	}
	stored, err := r.store.ListRuns(request.Context())
	if err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	allowed := make([]runResponse, 0, len(stored))
	for _, run := range stored {
		if r.allowedRun(run.ID) {
			allowed = append(allowed, newRunResponse(run))
		}
	}
	start := offset
	if start > len(allowed) {
		start = len(allowed)
	}
	end := start + limit
	if end > len(allowed) {
		end = len(allowed)
	}
	page := allowed[start:end]
	r.writeJSON(writer, http.StatusOK, runPageResponse{
		Runs: page,
		Page: paginationResponse{
			Offset: offset, Limit: limit, Returned: len(page), HasMore: end < len(allowed),
		},
	})
}

func parseRunPagination(values url.Values) (int, int, error) {
	for key, entries := range values {
		if (key != "offset" && key != "limit") || len(entries) != 1 {
			return 0, 0, errInvalidPagination
		}
	}
	offset := 0
	if rawOffset, present := values["offset"]; present {
		parsed, err := strconv.Atoi(rawOffset[0])
		if err != nil || parsed < 0 || parsed > maxRunPageOffset {
			return 0, 0, errInvalidPagination
		}
		offset = parsed
	}
	limit := defaultRunPageLimit
	if rawLimit, present := values["limit"]; present {
		parsed, err := strconv.Atoi(rawLimit[0])
		if err != nil || parsed < 1 || parsed > maxRunPageLimit {
			return 0, 0, errInvalidPagination
		}
		limit = parsed
	}
	return offset, limit, nil
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
