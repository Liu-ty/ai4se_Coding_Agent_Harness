package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/agent"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

func TestLocalCompositionPublishesProductionApprovalRequest(t *testing.T) {
	root, err := prepareRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	application, target, _, err := newLocalApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run, err := application.CreateRun(context.Background(), localRunRequest(root))
	if err != nil {
		t.Fatal(err)
	}
	var request agent.ApprovalRequired
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		events, listErr := target.ListEvents(context.Background(), run.ID, 1)
		if listErr != nil {
			t.Fatal(listErr)
		}
		for _, event := range events {
			if event.Type == "ApprovalRequired" {
				if err := json.Unmarshal(event.Payload, &request); err != nil {
					t.Fatal(err)
				}
			}
		}
		if request.Digest != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if request.Digest == "" || request.Action.Kind != "apply_patch" ||
		len(request.AffectedFiles) != 1 || request.AffectedFiles[0] != "a.txt" ||
		request.Risk == "" || request.RiskReason == "" ||
		len(request.BaselineEvidence) != 2 {
		t.Fatalf("production approval request = %#v", request)
	}
	if err := application.Approve(
		context.Background(), run.ID, string(request.Digest),
	); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		stored, getErr := target.GetRun(context.Background(), run.ID)
		if getErr == nil && stored.State == domain.StateSucceeded {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	stored, _ := target.GetRun(context.Background(), run.ID)
	t.Fatalf("approved production loop state = %s, want %s", stored.State, domain.StateSucceeded)
}
