package policy_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/policy"
)

func TestPatchProfileMatrix(t *testing.T) {
	action := domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"patch":"--- a/a.go\n+++ b/a.go\n@@ -0,0 +1 @@\n+package a\n"}`)}
	cases := []struct {
		profile domain.PermissionProfile
		want    policy.Verdict
	}{
		{domain.ProfileReview, policy.Deny},
		{domain.ProfileSupervised, policy.RequireApproval},
		{domain.ProfileWorkspaceAuto, policy.Allow},
	}
	for _, tc := range cases {
		got := policy.NewEngine().Evaluate(policy.Context{Profile: tc.profile}, action)
		if got.Verdict != tc.want {
			t.Fatalf("%s: got %s, want %s", tc.profile, got.Verdict, tc.want)
		}
	}
}

func TestHardDenialsPrecedeProfileMappings(t *testing.T) {
	cases := []struct {
		name   string
		action domain.Action
	}{
		{"unknown", domain.Action{Kind: "delete_file"}},
		{"raw shell", domain.Action{Kind: "shell", Args: json.RawMessage(`{"command":"go test ./..."}`)}},
		{"network", domain.Action{Kind: "network_request"}},
		{"credential", domain.Action{Kind: "read_file", Args: json.RawMessage(`{"path":".agent/credentials"}`)}},
		{"repository escape", domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"path":"../outside.go"}`)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto}, tc.action)
			if got.Verdict != policy.Deny || got.Risk != policy.RiskHardDenied {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestHardDeniesGitInternalAlternateSpellings(t *testing.T) {
	for _, path := range []string{"./.git/config", "dir/../.git/config", `.\\.git\\config`} {
		t.Run(path, func(t *testing.T) {
			action := domain.Action{Kind: "read_file", Args: json.RawMessage(`{"path":` + strconv.Quote(path) + `}`)}
			got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto}, action)
			if got.Verdict != policy.Deny || got.Risk != policy.RiskHardDenied || got.Code != "GIT_INTERNALS" {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestRunCheckRequiresConfiguredIdentifierOnly(t *testing.T) {
	engine := policy.NewEngine()
	ctx := policy.Context{Profile: domain.ProfileWorkspaceAuto, ConfiguredChecks: []string{"unit"}}
	for _, args := range []json.RawMessage{
		json.RawMessage(`{"command":"curl https://example.test"}`),
		json.RawMessage(`{"id":"unit","command":"curl https://example.test"}`),
		json.RawMessage(`{"id":"unknown"}`),
	} {
		got := engine.Evaluate(ctx, domain.Action{Kind: "run_check", Args: args})
		if got.Verdict != policy.Deny || got.Risk != policy.RiskHardDenied {
			t.Fatalf("args %s: got %#v", args, got)
		}
	}

	got := engine.Evaluate(ctx, domain.Action{Kind: "run_check", Args: json.RawMessage(`{"id":"unit"}`)})
	if got.Verdict != policy.Allow || got.Risk != policy.RiskNormal {
		t.Fatalf("configured check: got %#v", got)
	}
}

func TestHardDeniesKeyEndpointMismatch(t *testing.T) {
	action := domain.Action{Kind: "read_file", Args: json.RawMessage(`{"key_endpoint_mismatch":true}`)}
	got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto}, action)
	if got.Verdict != policy.Deny || got.Risk != policy.RiskHardDenied {
		t.Fatalf("got %#v", got)
	}
}

func TestMutationLimitsAreDerivedFromCanonicalPayload(t *testing.T) {
	largeContent, err := json.Marshal(map[string]string{"path": "a.go", "content": strings.Repeat("x", 1<<20+1)})
	if err != nil {
		t.Fatal(err)
	}
	largePatch, err := json.Marshal(map[string]string{"patch": "--- a/a.go\n+++ b/a.go\n" + strings.Repeat("+change\n", 501)})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []domain.Action{
		{Kind: "create_file", Args: largeContent},
		{Kind: "apply_patch", Args: largePatch},
	} {
		got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto}, action)
		if got.Verdict != policy.RequireApproval || got.Risk != policy.RiskGuarded {
			t.Fatalf("%s: got %#v", action.Kind, got)
		}
	}
}

func TestMutationRequiresCanonicalPayload(t *testing.T) {
	cases := []struct {
		name   string
		action domain.Action
	}{
		{"non-diff patch", domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"patch":"arbitrary text"}`)}},
		{"missing create content", domain.Action{Kind: "create_file", Args: json.RawMessage(`{"path":"a.go"}`)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto}, tc.action)
			if got.Verdict != policy.Deny || got.Risk != policy.RiskHardDenied {
				t.Fatalf("got %#v", got)
			}
		})
	}
}

func TestWorkspaceAutoRequiresApprovalForGuardedMutation(t *testing.T) {
	args, err := json.Marshal(map[string]string{"patch": "--- a/vendor/generated.go\n+++ b/vendor/generated.go\n" + strings.Repeat("+change\n", 501)})
	if err != nil {
		t.Fatal(err)
	}
	action := domain.Action{Kind: "apply_patch", Args: args}
	got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto}, action)
	if got.Verdict != policy.RequireApproval || got.Risk != policy.RiskGuarded {
		t.Fatalf("got %#v", got)
	}
}

func TestWorkspaceAutoDeniesDirtyMutation(t *testing.T) {
	action := domain.Action{Kind: "create_file", Args: json.RawMessage(`{"path":"a.go","content":"package a"}`)}
	got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileWorkspaceAuto, Dirty: true}, action)
	if got.Verdict != policy.Deny || got.Risk != policy.RiskGuarded {
		t.Fatalf("got %#v", got)
	}
}

func TestApprovalDigestBindsExactRequest(t *testing.T) {
	action := domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"path":"a.go","patch":"one"}`)}
	a := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"a.go": "one", "b.go": "two"})
	b := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"b.go": "two", "a.go": "one"})
	c := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"a.go": "changed", "b.go": "two"})
	d := policy.Digest("run-1", domain.ProfileWorkspaceAuto, action, map[string]string{"a.go": "one", "b.go": "two"})
	e := policy.Digest("run-1", domain.ProfileSupervised, domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"path":"a.go","patch":"two"}`)}, map[string]string{"a.go": "one", "b.go": "two"})

	if a == "" || a != b || a == c || a == d || a == e {
		t.Fatalf("unexpected digest binding: %q %q %q %q %q", a, b, c, d, e)
	}
}

func TestApprovalStoreConsumesGrantOnceAndOnlyForExactDigest(t *testing.T) {
	action := domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"path":"a.go"}`)}
	digest := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"a.go": "one"})
	changed := policy.Digest("run-1", domain.ProfileSupervised, action, map[string]string{"a.go": "two"})
	store := policy.NewApprovalStore()
	store.Grant(digest)

	if store.Consume(changed) {
		t.Fatal("changed baseline consumed an old approval")
	}
	if !store.Consume(digest) {
		t.Fatal("exact grant was not consumed")
	}
	if store.Consume(digest) {
		t.Fatal("approval was consumed more than once")
	}
}

func TestZeroValueApprovalStoreAcceptsGrant(t *testing.T) {
	var store policy.ApprovalStore
	digest := policy.Digest("run-1", domain.ProfileSupervised, domain.Action{Kind: "apply_patch"}, nil)
	store.Grant(digest)
	if !store.Consume(digest) {
		t.Fatal("zero-value store did not consume its grant")
	}
}

func TestEngineDoesNotAcceptRepositoryPolicyOverrides(t *testing.T) {
	action := domain.Action{Kind: "apply_patch", Args: json.RawMessage(`{"path":"a.go","policy":{"allow":true,"max_changed_lines":999999}}`)}
	got := policy.NewEngine().Evaluate(policy.Context{Profile: domain.ProfileReview}, action)
	if got.Verdict != policy.Deny {
		t.Fatalf("repository configuration weakened review policy: %#v", got)
	}
}
