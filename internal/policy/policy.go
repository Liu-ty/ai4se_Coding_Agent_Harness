// Package policy decides whether a canonical harness action may proceed.
package policy

import (
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

type Verdict string

const (
	Allow           Verdict = "ALLOW"
	RequireApproval Verdict = "REQUIRE_APPROVAL"
	Deny            Verdict = "DENY"
)

type Context struct {
	RunID            domain.RunID
	Profile          domain.PermissionProfile
	RepoRoot         string
	Dirty            bool
	Baselines        map[string]string
	ConfiguredChecks []string
}

type Decision struct {
	Verdict Verdict
	Risk    Risk
	Code    string
	Message string
	Digest  ApprovalDigest
}

type Engine struct{}

func NewEngine() Engine {
	return Engine{}
}

// Evaluate applies non-overridable risks before the selected permission profile.
func (Engine) Evaluate(ctx Context, action domain.Action) Decision {
	risk, code, message := classifyRisk(ctx, action)
	if risk == RiskHardDenied {
		return Decision{Verdict: Deny, Risk: risk, Code: code, Message: message}
	}

	mutation := isMutation(action.Kind)
	decision := Decision{Risk: risk, Code: code, Message: message}
	if ctx.Profile == domain.ProfileWorkspaceAuto && ctx.Dirty && mutation {
		decision.Verdict = Deny
		decision.Risk = RiskGuarded
		decision.Code = "DIRTY_WORKTREE"
		decision.Message = "workspace-auto mutations require a clean worktree"
		return decision
	}

	switch ctx.Profile {
	case domain.ProfileReview:
		if mutation || action.Kind == "run_check" {
			decision.Verdict = Deny
			decision.Code = "REVIEW_READ_ONLY"
			decision.Message = "review profile is read-only"
			return decision
		}
	case domain.ProfileSupervised:
		if mutation || action.Kind == "run_check" || risk == RiskGuarded {
			decision.Verdict = RequireApproval
			decision.Digest = Digest(ctx.RunID, ctx.Profile, action, ctx.Baselines)
			return decision
		}
	case domain.ProfileWorkspaceAuto:
		if risk == RiskGuarded {
			decision.Verdict = RequireApproval
			decision.Digest = Digest(ctx.RunID, ctx.Profile, action, ctx.Baselines)
			return decision
		}
	default:
		return Decision{Verdict: Deny, Risk: RiskHardDenied, Code: "UNKNOWN_PROFILE", Message: "permission profile is not recognized"}
	}

	decision.Verdict = Allow
	return decision
}

func isMutation(kind string) bool {
	return kind == "apply_patch" || kind == "create_file"
}
