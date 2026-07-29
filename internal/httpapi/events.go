package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func (r *Router) events(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	runID domain.RunID,
) {
	from, err := lastEventSequence(request.Header.Get("Last-Event-ID"))
	if err != nil {
		r.writeError(
			writer, http.StatusBadRequest, "INVALID_LAST_EVENT_ID",
			"Last-Event-ID is not a valid sequence", requestID,
		)
		return
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		r.writeError(writer, http.StatusInternalServerError, "SSE_UNSUPPORTED", "streaming is unavailable", requestID)
		return
	}
	initial, err := r.store.ListEvents(request.Context(), runID, from)
	if err != nil {
		r.mapError(writer, err, requestID)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.WriteHeader(http.StatusOK)

	next := from
	if !r.writeEvents(writer, flusher, initial, &next) {
		return
	}
	heartbeat := time.NewTicker(r.heartbeat)
	defer heartbeat.Stop()
	poll := time.NewTicker(r.poll)
	defer poll.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := writer.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-poll.C:
			events, listErr := r.store.ListEvents(request.Context(), runID, next)
			if listErr != nil {
				if errors.Is(listErr, context.Canceled) || errors.Is(listErr, context.DeadlineExceeded) {
					return
				}
				return
			}
			if !r.writeEvents(writer, flusher, events, &next) {
				return
			}
		}
	}
}

func lastEventSequence(value string) (uint64, error) {
	if strings.TrimSpace(value) == "" {
		return 1, nil
	}
	sequence, err := strconv.ParseUint(value, 10, 64)
	if err != nil || sequence == ^uint64(0) {
		return 0, errors.New("invalid sequence")
	}
	return sequence + 1, nil
}

func (r *Router) writeEvents(
	writer http.ResponseWriter,
	flusher http.Flusher,
	events []domain.RunEvent,
	next *uint64,
) bool {
	for _, event := range events {
		data := event.Payload
		if !json.Valid(data) {
			data = json.RawMessage(`{"code":"INVALID_EVENT_PAYLOAD"}`)
		}
		redacted := r.redact(string(data))
		if !json.Valid([]byte(redacted)) {
			redacted = `{"code":"REDACTED_EVENT"}`
		}
		frame := fmt.Sprintf(
			"id: %d\nevent: %s\ndata: %s\n\n",
			event.Sequence, safeEventType(r.redact(event.Type)), redacted,
		)
		if _, err := writer.Write([]byte(frame)); err != nil {
			return false
		}
		flusher.Flush()
		if event.Sequence == ^uint64(0) {
			return false
		}
		*next = event.Sequence + 1
	}
	return true
}

func safeEventType(value string) string {
	value = strings.Map(func(char rune) rune {
		if char == '\r' || char == '\n' || char < 0x20 || char == 0x7f {
			return -1
		}
		return char
	}, value)
	if value == "" {
		return "RunEvent"
	}
	return value
}
