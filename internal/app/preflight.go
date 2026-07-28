package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/config"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/credentials"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/workspace"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

const (
	CodeGitMissing                   = "GIT_MISSING"
	CodeNotGitRepository             = "NOT_GIT_REPOSITORY"
	CodeBaselineUnavailable          = "BASELINE_UNAVAILABLE"
	CodeBaselineTooLarge             = "BASELINE_TOO_LARGE"
	CodeInvalidConfig                = "INVALID_CONFIG"
	CodeExecutableMissing            = "EXECUTABLE_MISSING"
	CodeInvalidEndpoint              = "INVALID_ENDPOINT"
	CodeEndpointConfirmationRequired = "CUSTOM_ENDPOINT_CONFIRMATION_REQUIRED"
	CodeCredentialMissing            = "CREDENTIAL_NOT_CONFIGURED"
	CodeCredentialStoreUnavailable   = "CREDENTIAL_STORE_UNAVAILABLE"
	CodeCredentialEndpointMismatch   = "CREDENTIAL_ENDPOINT_MISMATCH"
	CodeInvalidProfile               = "INVALID_PROFILE"
	CodeDirtyWorktree                = "DIRTY_WORKTREE"
	CodeDirtyWorktreeApproval        = "DIRTY_WORKTREE_APPROVAL_REQUIRED"
	CodeDirtyWorktreeReadOnly        = "DIRTY_WORKTREE_READ_ONLY"
	CodeDataDirectoryUnavailable     = "DATA_DIRECTORY_UNAVAILABLE"
)

type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

type PreflightReport struct {
	OK               bool      `json:"ok"`
	Findings         []Finding `json:"findings"`
	RepoRoot         string    `json:"repo_root"`
	BaselineCommit   string    `json:"baseline_commit"`
	BaselineDiffHash string    `json:"baseline_diff_hash"`
}

type preflightResult struct {
	report PreflightReport
	config config.Config
}

func (s *Service) Preflight(ctx context.Context, request CreateRunRequest) PreflightReport {
	return s.preflight(ctx, request).report
}

func (s *Service) preflight(ctx context.Context, request CreateRunRequest) preflightResult {
	result := preflightResult{report: PreflightReport{Findings: make([]Finding, 0)}}
	add := func(code string, severity Severity, message string) {
		result.report.Findings = append(result.report.Findings, Finding{
			Code: code, Severity: severity, Message: message,
		})
	}

	if !validProfile(request.Profile) {
		add(CodeInvalidProfile, SeverityError, "permission profile is not recognized")
	}

	gitPath, err := s.lookPath("git")
	if err != nil {
		add(CodeGitMissing, SeverityError, "Git executable is not available")
	} else {
		root, rootErr := canonicalGitRoot(ctx, gitPath, request.RepoRoot)
		if rootErr != nil {
			add(CodeNotGitRepository, SeverityError, "path is not a Git worktree")
		} else {
			result.report.RepoRoot = root
			commit, status, snapshotHash, baselineErr := gitBaseline(ctx, gitPath, root, s.maxBaselineBytes)
			if baselineErr != nil {
				if errors.Is(baselineErr, errBaselineTooLarge) {
					add(CodeBaselineTooLarge, SeverityError, "repository baseline exceeds the configured capture limit")
				} else {
					add(CodeBaselineUnavailable, SeverityError, "repository baseline could not be captured")
				}
			} else {
				result.report.BaselineCommit = commit
				result.report.BaselineDiffHash = snapshotHash
				switch {
				case status == "":
				case request.Profile == domain.ProfileWorkspaceAuto:
					add(CodeDirtyWorktree, SeverityError, "workspace-auto requires a clean worktree")
				case request.Profile == domain.ProfileSupervised:
					add(CodeDirtyWorktreeApproval, SeverityWarning, "dirty supervised workspace requires explicit approval")
				case request.Profile == domain.ProfileReview:
					add(CodeDirtyWorktreeReadOnly, SeverityInfo, "dirty workspace will be reviewed without mutation")
				}
			}

			cfg, configErr := loadProjectConfig(request.ConfigPath, root)
			if configErr != nil {
				add(CodeInvalidConfig, SeverityError, "project configuration is invalid")
			} else {
				result.config = cfg
				for _, stage := range cfg.Validation {
					spec, resolveErr := config.ResolveStage(stage, runtime.GOOS)
					if resolveErr != nil {
						add(CodeInvalidConfig, SeverityError, "validation stage cannot be resolved")
						continue
					}
					if _, resolveErr = workspace.Resolve(root, spec.WorkingDirectory); resolveErr != nil {
						add(CodeInvalidConfig, SeverityError, "validation working directory is outside the repository")
						continue
					}
					if _, executableErr := s.lookPath(spec.Executable); executableErr != nil && spec.Required {
						add(CodeExecutableMissing, SeverityError, "required validation executable is not available")
					}
				}
			}
		}
	}

	ref, endpointErr := credentialRef(request.Provider, request.Endpoint)
	if endpointErr != nil {
		add(CodeInvalidEndpoint, SeverityError, "provider endpoint is invalid")
	} else if !knownCredentialEndpoint(request.Provider, ref.Host) && !request.ConfirmCustomEndpoint {
		add(CodeEndpointConfirmationRequired, SeverityError, "custom provider endpoint requires explicit confirmation")
	} else if s.creds == nil {
		add(CodeCredentialStoreUnavailable, SeverityError, "credential storage is unavailable")
	} else {
		status, statusErr := s.creds.Status(ctx, ref)
		switch {
		case statusErr != nil:
			add(CodeCredentialStoreUnavailable, SeverityError, "credential storage is unavailable")
		case !status.Configured:
			add(CodeCredentialMissing, SeverityError, "credential is not configured for this provider endpoint")
		case !sameCredentialRef(status.Ref, ref):
			add(CodeCredentialEndpointMismatch, SeverityError, "credential is bound to a different provider endpoint")
		}
	}

	if err := s.probeWritable(s.dataDir); err != nil {
		add(CodeDataDirectoryUnavailable, SeverityError, "application data directory is not writable")
	}

	result.report.OK = true
	for _, finding := range result.report.Findings {
		if finding.Severity == SeverityError {
			result.report.OK = false
			break
		}
	}
	return result
}

func validProfile(profile domain.PermissionProfile) bool {
	return profile == domain.ProfileReview ||
		profile == domain.ProfileSupervised ||
		profile == domain.ProfileWorkspaceAuto
}

func canonicalGitRoot(ctx context.Context, gitPath, requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return "", errors.New("repository path is empty")
	}
	resolved, err := workspace.ResolveRoot(requested)
	if err != nil {
		return "", err
	}
	output, err := gitCommand(ctx, gitPath, resolved, "rev-parse", "--show-toplevel")
	if err != nil || output == "" {
		return "", errors.New("not a Git worktree")
	}
	root, err := workspace.ResolveRoot(filepath.Clean(output))
	if err != nil {
		return "", err
	}
	return filepath.Clean(root), nil
}

var errBaselineTooLarge = errors.New("repository baseline exceeds limit")

func gitBaseline(ctx context.Context, gitPath, root string, maxBytes int64) (string, string, string, error) {
	if maxBytes <= 0 {
		return "", "", "", errBaselineTooLarge
	}
	commit, err := gitCommand(ctx, gitPath, root, "rev-parse", "HEAD")
	if err != nil {
		return "", "", "", err
	}
	remaining := maxBytes
	readGit := func(args ...string) ([]byte, error) {
		output, commandErr := gitCommandLimited(ctx, gitPath, root, remaining, args...)
		if commandErr != nil {
			return nil, commandErr
		}
		remaining -= int64(len(output))
		return output, nil
	}
	statusBytes, err := readGit("status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return "", "", "", err
	}
	diff, err := readGit("diff", "--no-ext-diff", "--binary", "HEAD", "--")
	if err != nil {
		return "", "", "", err
	}
	untrackedPaths, err := readGit("ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", "", "", err
	}
	snapshot := sha256.New()
	_, _ = snapshot.Write(statusBytes)
	_, _ = snapshot.Write([]byte{0})
	_, _ = snapshot.Write(diff)
	_, _ = snapshot.Write([]byte{0})
	for _, path := range bytes.Split(untrackedPaths, []byte{0}) {
		if len(path) == 0 {
			continue
		}
		relative := filepath.Clean(filepath.FromSlash(string(path)))
		candidate, resolveErr := workspace.Resolve(root, relative)
		if resolveErr != nil {
			return "", "", "", resolveErr
		}
		info, statErr := os.Lstat(candidate)
		if statErr != nil {
			return "", "", "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", "", "", errors.New("untracked baseline path is not a regular file")
		}
		file, openErr := os.Open(candidate)
		if openErr != nil {
			return "", "", "", openErr
		}
		openedInfo, openedStatErr := file.Stat()
		if openedStatErr != nil || !os.SameFile(info, openedInfo) {
			_ = file.Close()
			if openedStatErr != nil {
				return "", "", "", openedStatErr
			}
			return "", "", "", errors.New("untracked baseline path changed during capture")
		}
		_, _ = snapshot.Write(path)
		_, _ = snapshot.Write([]byte{0})
		written, copyErr := io.Copy(snapshot, io.LimitReader(file, remaining+1))
		closeErr := file.Close()
		if copyErr != nil {
			return "", "", "", copyErr
		}
		if closeErr != nil {
			return "", "", "", closeErr
		}
		if written > remaining {
			return "", "", "", errBaselineTooLarge
		}
		remaining -= written
		_, _ = snapshot.Write([]byte{0})
	}
	return commit, strings.TrimSpace(string(statusBytes)), hex.EncodeToString(snapshot.Sum(nil)), nil
}

func gitCommand(ctx context.Context, gitPath, root string, args ...string) (string, error) {
	output, err := gitCommandBytes(ctx, gitPath, root, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitCommandBytes(ctx context.Context, gitPath, root string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, gitPath, args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func gitCommandLimited(
	ctx context.Context,
	gitPath string,
	root string,
	maxBytes int64,
	args ...string,
) ([]byte, error) {
	if maxBytes < 0 {
		return nil, errBaselineTooLarge
	}
	command := exec.CommandContext(ctx, gitPath, args...)
	command.Dir = root
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	output, readErr := io.ReadAll(io.LimitReader(stdout, maxBytes+1))
	if int64(len(output)) > maxBytes {
		_ = command.Process.Kill()
		_ = command.Wait()
		return nil, errBaselineTooLarge
	}
	waitErr := command.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		return nil, waitErr
	}
	return output, nil
}

func loadProjectConfig(configPath, root string) (config.Config, error) {
	var path string
	if configPath == "" {
		path = filepath.Join(root, ".ai4se-harness.toml")
	} else {
		if filepath.IsAbs(configPath) {
			return config.Config{}, errors.New("configuration path must be repository-relative")
		}
		resolved, err := workspace.Resolve(root, configPath)
		if err != nil {
			return config.Config{}, err
		}
		path = resolved
	}
	file, err := os.Open(path)
	if err != nil {
		return config.Config{}, err
	}
	defer func() { _ = file.Close() }()
	return config.Load(file)
}

func credentialRef(providerID, endpoint string) (credentials.Ref, error) {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || !safeCredentialEndpoint(parsed) ||
		parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" {
		return credentials.Ref{}, errors.New("invalid endpoint")
	}
	return credentials.Ref{
		Provider: strings.ToLower(strings.TrimSpace(providerID)),
		Host:     strings.ToLower(strings.TrimSuffix(parsed.Host, ".")),
	}, nil
}

func safeCredentialEndpoint(parsed *url.URL) bool {
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	return net.ParseIP(parsed.Hostname()) != nil && net.ParseIP(parsed.Hostname()).IsLoopback()
}

func knownCredentialEndpoint(providerID, host string) bool {
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return (providerID == "openai" && host == "api.openai.com") ||
		(providerID == "anthropic" && host == "api.anthropic.com")
}

func sameCredentialRef(left, right credentials.Ref) bool {
	return strings.EqualFold(left.Provider, right.Provider) &&
		strings.EqualFold(strings.TrimSuffix(left.Host, "."), strings.TrimSuffix(right.Host, "."))
}

func probeWritableDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("data directory is empty")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(path, ".writable-*")
	if err != nil {
		return err
	}
	name := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(name)
	if closeErr != nil {
		return fmt.Errorf("close writability probe: %w", closeErr)
	}
	if removeErr != nil {
		return fmt.Errorf("remove writability probe: %w", removeErr)
	}
	return nil
}
