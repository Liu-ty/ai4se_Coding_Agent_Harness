import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { ApiError } from "../api/client";
import { Credentials } from "./Credentials";
import { DemoGallery } from "./DemoGallery";
import { NewRun } from "./NewRun";
import type { PreflightReport } from "../api/types";

it("loads non-secret credential status, distinguishes update, and clears secret immediately", async () => {
  let resolveSubmit!: () => void;
  const submit = vi.fn(() => new Promise<void>((resolve) => { resolveSubmit = resolve; }));
  const loadStatus = vi.fn().mockResolvedValue({
    ref: { provider: "openai", host: "api.openai.com" },
    configured: true, backend: "keyring", updated_at: "2026-07-29T00:00:00Z",
  });
  render(<Credentials loadStatus={loadStatus} onSave={submit} onClear={vi.fn()} />);
  expect(await screen.findByText(/configured in keyring/i)).toBeVisible();
  const input = screen.getByLabelText("Secret");
  await userEvent.type(input, "canary-value");
  await userEvent.click(screen.getByRole("button", { name: "Update credential" }));
  expect(input).toHaveValue("");
  expect(screen.queryByText("canary-value")).toBeNull();
  resolveSubmit();
});

it("credential clear exposes pending failure and recovery", async () => {
  const onClear = vi.fn().mockRejectedValueOnce(new Error("locked")).mockResolvedValueOnce(undefined);
  render(<Credentials loadStatus={vi.fn().mockResolvedValue({
    ref: { provider: "openai", host: "api.openai.com" }, configured: true,
    backend: "keyring", updated_at: "2026-07-29T00:00:00Z",
  })} onSave={vi.fn()} onClear={onClear} />);
  const clear = await screen.findByRole("button", { name: "Clear credential" });
  await userEvent.click(clear);
  expect(await screen.findByText(/could not be cleared/i)).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "Retry clear" }));
  expect(await screen.findByText(/credential cleared/i)).toBeVisible();
});

it("credential status treats only CREDENTIAL_NOT_FOUND as unconfigured", async () => {
  const loadStatus = vi.fn()
    .mockRejectedValueOnce(new ApiError("SERVER_ERROR", "vault unavailable"))
    .mockResolvedValueOnce({
      ref: { provider: "openai", host: "api.openai.com" }, configured: false,
      backend: "", updated_at: "",
    });
  render(<Credentials loadStatus={loadStatus} onSave={vi.fn()} onClear={vi.fn()} />);
  expect(await screen.findByRole("alert")).toHaveTextContent("vault unavailable");
  expect(screen.queryByText("Not configured", { exact: true })).toBeNull();
  await userEvent.click(screen.getByRole("button", { name: "Retry status" }));
  expect(await screen.findByText(/add a credential/i)).toBeVisible();
});

it("credential status normalizes identity and ignores an aborted stale response", async () => {
  const pending: Array<{
    provider: string;
    host: string;
    signal?: AbortSignal;
    resolve: (status: {
      ref: { provider: string; host: string };
      configured: boolean;
      backend: string;
      updated_at: string;
    }) => void;
  }> = [];
  const loadStatus = vi.fn((provider: string, host: string, signal?: AbortSignal) =>
    new Promise<{
      ref: { provider: string; host: string };
      configured: boolean;
      backend: string;
      updated_at: string;
    }>((resolve) => pending.push({ provider, host, signal, resolve })));
  render(<Credentials loadStatus={loadStatus} onSave={vi.fn()} onClear={vi.fn()} />);
  await vi.waitFor(() => expect(pending).toHaveLength(1));
  fireEvent.change(screen.getByLabelText("Provider"), { target: { value: " ANTHROPIC " } });
  await vi.waitFor(() => expect(pending).toHaveLength(2));
  expect(pending[0].signal?.aborted).toBe(true);
  expect(pending[1]).toMatchObject({ provider: "anthropic", host: "api.openai.com" });
  pending[1].resolve({
    ref: { provider: "anthropic", host: "api.openai.com" }, configured: true,
    backend: "new-backend", updated_at: "2026-07-29T00:00:00Z",
  });
  expect(await screen.findByText(/configured in new-backend/i)).toBeVisible();
  pending[0].resolve({
    ref: { provider: "openai", host: "api.openai.com" }, configured: true,
    backend: "stale-backend", updated_at: "2026-07-28T00:00:00Z",
  });
  await vi.waitFor(() => expect(screen.queryByText(/stale-backend/i)).toBeNull());
});

it("credential not-found response maps to the unconfigured state", async () => {
  render(<Credentials
    loadStatus={vi.fn().mockRejectedValue(new ApiError(
      "CREDENTIAL_NOT_FOUND", "credential was not found",
    ))}
    onSave={vi.fn()} onClear={vi.fn()} />);
  expect(await screen.findByText(/add a credential/i)).toBeVisible();
});

it("does not let a save completion for an old identity replace the current status", async () => {
  let resolveSave!: () => void;
  const onSave = vi.fn(() => new Promise<void>((resolve) => { resolveSave = resolve; }));
  const loadStatus = vi.fn(async (provider: string, host: string) => ({
    ref: { provider, host },
    configured: true,
    backend: provider === "anthropic" ? "new-backend" : "stale-backend",
    updated_at: "2026-07-29T00:00:00Z",
  }));
  render(<Credentials loadStatus={loadStatus} onSave={onSave} onClear={vi.fn()} />);
  expect(await screen.findByText(/configured in stale-backend/i)).toBeVisible();
  await userEvent.type(screen.getByLabelText("Secret"), "canary-value");
  await userEvent.click(screen.getByRole("button", { name: "Update credential" }));
  fireEvent.change(screen.getByLabelText("Provider"), { target: { value: " ANTHROPIC " } });
  expect(await screen.findByText(/configured in new-backend/i)).toBeVisible();
  await act(async () => resolveSave());
  expect(screen.getByText(/configured in new-backend/i)).toBeVisible();
  expect(screen.queryByText(/configured in stale-backend/i)).toBeNull();
});

it("does not let a clear completion for an old identity clear the current status", async () => {
  let resolveClear!: () => void;
  const onClear = vi.fn(() => new Promise<void>((resolve) => { resolveClear = resolve; }));
  const loadStatus = vi.fn(async (provider: string, host: string) => ({
    ref: { provider, host },
    configured: true,
    backend: provider === "anthropic" ? "new-backend" : "old-backend",
    updated_at: "2026-07-29T00:00:00Z",
  }));
  render(<Credentials loadStatus={loadStatus} onSave={vi.fn()} onClear={onClear} />);
  await userEvent.click(await screen.findByRole("button", { name: "Clear credential" }));
  fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "anthropic" } });
  expect(await screen.findByText(/configured in new-backend/i)).toBeVisible();
  await act(async () => resolveClear());
  expect(screen.getByText(/configured in new-backend/i)).toBeVisible();
  expect(screen.queryByText("Not configured", { exact: true })).toBeNull();
});

it("demo gallery labels every scenario and hides unavailable controls", () => {
  render(<DemoGallery fixedRuns={["feedback-loop"]} onOpen={vi.fn()} />);
  expect(screen.getAllByText("SIMULATED").length).toBeGreaterThan(1);
  expect(screen.queryByRole("button", { name: /new run/i })).toBeNull();
  expect(screen.queryByRole("link", { name: /credentials/i })).toBeNull();
});

it("gives each simulated scenario a run-specific accessible action", () => {
  render(<DemoGallery fixedRuns={["feedback-loop", "policy-denial"]} onOpen={vi.fn()} />);
  expect(screen.getByRole("button", { name: "Open SIMULATED feedback-loop demo" })).toBeVisible();
  expect(screen.getByRole("button", { name: "Open SIMULATED policy-denial demo" })).toBeVisible();
});

it("includes explicit custom-endpoint confirmation in preflight requests", async () => {
  const onPreflight = vi.fn().mockResolvedValue({
    ok: false, findings: [], repo_root: "C:\\repo", baseline_commit: "", baseline_diff_hash: "",
  });
  render(<NewRun onPreflight={onPreflight} onCreate={vi.fn()} />);
  fireEvent.change(screen.getByLabelText("Repository path"), { target: { value: "C:\\repo" } });
  fireEvent.change(screen.getByLabelText("Task description"), { target: { value: "Repair" } });
  fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "openai" } });
  fireEvent.change(screen.getByLabelText("Model"), { target: { value: "gpt-test" } });
  fireEvent.change(screen.getByLabelText("Endpoint"), {
    target: { value: "https://gateway.example.test/v1/chat/completions" },
  });
  await userEvent.click(screen.getByLabelText("Confirm this custom endpoint"));
  await userEvent.click(screen.getByRole("button", { name: "Validate preflight" }));
  expect(onPreflight).toHaveBeenCalledWith(expect.objectContaining({
    endpoint: "https://gateway.example.test/v1/chat/completions",
    confirm_custom_endpoint: true,
  }), expect.any(AbortSignal));
});

it("supports a keyboard-only supervised run draft", async () => {
  const onCreate = vi.fn();
  const report: PreflightReport = {
    ok: true, findings: [{ code: "REPOSITORY_REACHABLE", severity: "info", message: "Repository reachable" }],
    repo_root: "C:\\repo", baseline_commit: "abc", baseline_diff_hash: "def",
  };
  const onPreflight = vi.fn().mockResolvedValue(report);
  const user = userEvent.setup();
  render(<NewRun onPreflight={onPreflight} onCreate={onCreate} />);
  await user.tab();
  await user.keyboard("C:\\repo");
  await user.tab();
  await user.keyboard("Repair the failing check");
  await user.tab();
  await user.keyboard("mock");
  await user.tab();
  await user.keyboard("mock-v1");
  await user.tab();
  await user.tab();
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Validate preflight" }));
  await user.keyboard("{Enter}");
  expect(await screen.findByText("Repository reachable")).toBeVisible();
  const create = screen.getByRole("button", { name: "Create run" });
  expect(create).toBeEnabled();
  create.focus();
  await user.keyboard("{Enter}");
  expect(onCreate).toHaveBeenCalledWith(expect.objectContaining({ profile: "supervised" }));
});

it("blocks creation until the current draft passes preflight", async () => {
  render(<NewRun onPreflight={vi.fn()} onCreate={vi.fn()} />);
  expect(screen.getByRole("button", { name: "Create run" })).toBeDisabled();
});

it("aborts edited preflight and ignores its late out-of-order success", async () => {
  const pending: Array<{
    signal?: AbortSignal;
    resolve: (report: PreflightReport) => void;
  }> = [];
  const onPreflight = vi.fn((_request, signal?: AbortSignal) =>
    new Promise<PreflightReport>((resolve) => pending.push({ signal, resolve })));
  render(<NewRun onPreflight={onPreflight} onCreate={vi.fn()} />);
  fireEvent.change(screen.getByLabelText("Repository path"), { target: { value: "C:\\repo" } });
  fireEvent.change(screen.getByLabelText("Task description"), { target: { value: "First task" } });
  fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "mock" } });
  fireEvent.change(screen.getByLabelText("Model"), { target: { value: "mock-v1" } });
  await userEvent.click(screen.getByRole("button", { name: "Validate preflight" }));
  fireEvent.change(screen.getByLabelText("Task description"), { target: { value: "Second task" } });
  expect(pending[0].signal?.aborted).toBe(true);
  await userEvent.click(screen.getByRole("button", { name: "Validate preflight" }));
  pending[1].resolve({
    ok: false, findings: [{ code: "BLOCKED", severity: "error", message: "Second is blocked" }],
    repo_root: "C:\\repo", baseline_commit: "new", baseline_diff_hash: "new",
  });
  expect(await screen.findByText("Second is blocked")).toBeVisible();
  pending[0].resolve({
    ok: true, findings: [], repo_root: "C:\\repo",
    baseline_commit: "stale", baseline_diff_hash: "stale",
  });
  await vi.waitFor(() =>
    expect(screen.getByRole("button", { name: "Create run" })).toBeDisabled());
  expect(screen.queryByText(/preflight passed/i)).toBeNull();
});
