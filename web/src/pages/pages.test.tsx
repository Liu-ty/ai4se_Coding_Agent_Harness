import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
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

it("demo gallery labels every scenario and hides unavailable controls", () => {
  render(<DemoGallery fixedRuns={["feedback-loop"]} onOpen={vi.fn()} />);
  expect(screen.getAllByText("SIMULATED").length).toBeGreaterThan(1);
  expect(screen.queryByRole("button", { name: /new run/i })).toBeNull();
  expect(screen.queryByRole("link", { name: /credentials/i })).toBeNull();
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
