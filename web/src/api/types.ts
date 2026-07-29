export type PermissionProfile = "review" | "supervised" | "workspace-auto";

export interface Run {
  id: string;
  state: string;
  profile: PermissionProfile;
  task: string;
  repo_root: string;
  current_stage: string;
  created_at: string;
  updated_at: string;
}

export interface CreateRunRequest {
  repo_root: string;
  task: string;
  provider: string;
  model: string;
  endpoint: string;
  confirm_custom_endpoint: boolean;
  profile: PermissionProfile;
  config_path?: string;
}

export interface Finding {
  code: string;
  severity: "info" | "warning" | "error";
  message: string;
}

export interface PreflightReport {
  ok: boolean;
  findings: Finding[];
  repo_root: string;
  baseline_commit: string;
  baseline_diff_hash: string;
}

export interface Artifact {
  id: string;
  run_id: string;
  kind: string;
  sha256: string;
  content: string;
  truncated: boolean;
}

export interface CredentialStatus {
  ref: { provider: string; host: string };
  configured: boolean;
  backend: string;
  updated_at: string;
}

export interface ErrorEnvelope {
  error: { code: string; message: string; request_id: string };
}

export type ConnectionState =
  | "connected"
  | "disconnected"
  | "reconnecting"
  | "reconnected";

export interface RunEvent {
  sequence: number;
  type: string;
  at?: string;
  payload: Record<string, unknown>;
}

export interface RuntimeCapabilities {
  createRuns: boolean;
  cancelRuns: boolean;
  approvals: boolean;
  artifacts: boolean;
  configValidation: boolean;
  credentials: boolean;
  demo: boolean;
  fixedRuns: string[];
}

export interface RuntimeConfig {
  csrfToken?: string;
  capabilities: RuntimeCapabilities;
}
