import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { ApprovalPanel } from "./ApprovalPanel";
import { DiffViewer } from "./DiffViewer";
import { Timeline } from "./Timeline";
import { RunDetail } from "../pages/RunDetail";
import { Dashboard } from "../pages/Dashboard";
import type { Run } from "../api/types";

const fixtureEvents = [{
  sequence: 8,
  type: "FeedbackProduced",
  at: "2026-07-29T08:00:00Z",
  payload: {
    category: "TEST_FAILURE",
    summary: "Expected 2, received 1",
    evidence: [{ source: "stderr", message: "REDACTED", path: "sum.test.ts", line: 12 }],
    budgets: { mutations: { used: 1, limit: 5 } },
  },
}];

it("renders validation failure evidence and remaining budgets", async () => {
  render(<Timeline events={fixtureEvents} />);
  expect(screen.getByText("TEST_FAILURE")).toBeVisible();
  expect(screen.getByText("Mutations 1 / 5")).toBeVisible();
  expect(screen.queryByText("REDACTED")).toBeNull();
  await userEvent.click(screen.getByRole("button", { name: /show redacted evidence/i }));
  expect(screen.getByText("REDACTED")).toBeVisible();
});

it("approval panel offers exactly one-time approve and reject decisions", async () => {
  const onDecision = vi.fn();
  render(<ApprovalPanel request={{
    digest: "sha256:abc", action: { kind: "apply_patch", args: { patch: "bounded" } },
    affectedFiles: ["src/sum.ts"], risk: "GUARDED", riskReason: "Modifies one tracked file",
  }} onDecision={onDecision} />);
  await userEvent.click(screen.getByRole("button", { name: "Approve once" }));
  expect(onDecision).toHaveBeenCalledWith("approve", "sha256:abc");
  expect(screen.queryByText(/always allow|remember/i)).toBeNull();
  expect(screen.getByRole("button", { name: "Approve once" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Reject" })).toBeDisabled();
});

it.each(["approve", "reject"] as const)(
  "approval panel recovers when %s fails",
  async (decision) => {
    const onDecision = vi.fn().mockRejectedValue(new Error(`${decision} unavailable`));
    render(<ApprovalPanel request={{
      digest: "sha256:abc",
      action: { kind: "apply_patch", args: { patch: "bounded" } },
      affectedFiles: ["src/sum.ts"],
      risk: "GUARDED",
      riskReason: "Modifies one tracked file",
    }} onDecision={onDecision} />);
    await userEvent.click(screen.getByRole("button", {
      name: decision === "approve" ? "Approve once" : "Reject",
    }));
    expect(await screen.findByRole("alert")).toHaveTextContent(`${decision} unavailable`);
    expect(screen.getByRole("button", { name: "Approve once" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Reject" })).toBeEnabled();
  },
);

it("approval panel waits for the decision promise", async () => {
  let resolve!: () => void;
  const onDecision = vi.fn(() => new Promise<void>((done) => { resolve = done; }));
  render(<ApprovalPanel request={{
    digest: "sha256:abc",
    action: { kind: "apply_patch", args: { patch: "bounded" } },
    affectedFiles: ["src/sum.ts"],
    risk: "GUARDED",
    riskReason: "Modifies one tracked file",
  }} onDecision={onDecision} />);
  await userEvent.click(screen.getByRole("button", { name: "Approve once" }));
  expect(screen.getByRole("button", { name: "Approving…" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Reject" })).toBeDisabled();
  resolve();
});

it("renders a read-only keyboard-focusable diff", () => {
  render(<DiffViewer content={"- return 1\n+ return 2"} truncated={false} />);
  expect(screen.getByRole("region", { name: "Read-only diff" })).toHaveAttribute("tabindex", "0");
  expect(screen.queryByRole("textbox")).toBeNull();
});

it("renders server timestamps, unavailable timestamps, truncation, and SIMULATED on each demo row", () => {
  render(<Timeline simulated events={[
    { sequence: 1, type: "PolicyEvaluated", at: "2026-07-29T01:02:03Z",
      payload: { summary: "Denied", output_truncated: true } },
    { sequence: 2, type: "FeedbackProduced", payload: { summary: "Retry" } },
  ]} />);
  expect(screen.getByText(/2026-07-29 01:02:03/i)).toBeVisible();
  expect(screen.getByText("Timestamp unavailable")).toBeVisible();
  expect(screen.getByText(/server truncated this event output/i)).toBeVisible();
  expect(screen.getAllByText("SIMULATED")).toHaveLength(2);
});

it("formats fractional timestamps consistently and exposes disclosure state", async () => {
  render(<Timeline events={[{
    sequence: 3,
    type: "FeedbackProduced",
    at: "2026-07-29T01:02:03.456Z",
    payload: { evidence: [{ source: "stderr", message: "bounded" }] },
  }]} />);
  expect(screen.getByText("2026-07-29 01:02:03 UTC")).toBeVisible();
  const disclosure = screen.getByRole("button", { name: "Show redacted evidence" });
  expect(disclosure).toHaveAttribute("type", "button");
  expect(disclosure).toHaveAttribute("aria-expanded", "false");
  await userEvent.click(disclosure);
  expect(disclosure).toHaveAttribute("aria-expanded", "true");
});

it("derives budgets and terminal reason only from server events", () => {
  const run: Run = {
    id: "run-1", state: "SUCCEEDED", profile: "supervised" as const, task: "Repair",
    repo_root: "C:\\repo", current_stage: "final", created_at: "", updated_at: "",
  };
  const { rerender } = render(<RunDetail run={run} events={[{
    sequence: 4, type: "BudgetUpdated", payload: {
      budgets: { decisions: { used: 3, limit: 30 }, mutations: { used: 2, limit: 5 } },
      reason: "ALL_REQUIRED_CHECKS_PASSED",
    },
  }]} connection="connected" />);
  expect(screen.getByText("Decisions 3 / 30")).toBeVisible();
  expect(screen.getAllByText("Mutations 2 / 5")).toHaveLength(2);
  expect(screen.getByText("ALL_REQUIRED_CHECKS_PASSED")).toBeVisible();
  rerender(<RunDetail run={run} events={[]} connection="connected" />);
  expect(screen.getByText("Budget data unavailable")).toBeVisible();
  expect(screen.getByText("Terminal reason unavailable")).toBeVisible();
});

it("offers cursor-preserving reconnect after the event stream fails", async () => {
  const reconnect = vi.fn();
  render(<RunDetail run={{
    id: "run-1", state: "DECIDING", profile: "supervised", task: "Repair",
    repo_root: "C:\\repo", current_stage: "decision", created_at: "", updated_at: "",
  }} events={fixtureEvents} connection="failed"
    streamError={{ kind: "connect", message: "Event stream unavailable", attempts: 5, lastSequence: 8 }}
    onReconnect={reconnect} />);
  expect(screen.getByRole("alert")).toHaveTextContent("Event stream unavailable");
  await userEvent.click(screen.getByRole("button", { name: "Reconnect event stream" }));
  expect(reconnect).toHaveBeenCalledOnce();
});

it("derives detail facts from sequence order and marks stopped runs as danger", () => {
  render(<RunDetail run={{
    id: "run-stopped", state: "STOPPED", profile: "supervised", task: "Repair",
    repo_root: "C:\\repo", current_stage: "final", created_at: "", updated_at: "",
  }} events={[
    { sequence: 9, type: "RunStopped", payload: {
      reason: "USER_CANCELLED", budgets: { decisions: { used: 4, limit: 30 } },
    } },
    { sequence: 2, type: "BudgetUpdated", payload: {
      reason: "STALE", budgets: { decisions: { used: 1, limit: 30 } },
    } },
  ]} connection="connected" />);
  expect(screen.getByText("USER_CANCELLED")).toBeVisible();
  expect(screen.getByText("Decisions 4 / 30")).toBeVisible();
  expect(screen.getByText("Latest sequence: 9. Reconnect resumes from this cursor.")).toBeVisible();
  expect(screen.getByText("STOPPED")).toHaveClass("status-danger");
});

it("dashboard renders four server-derived KPIs, visualization, activity, and runs", () => {
  const runs: Run[] = [{
    id: "run-1", state: "SUCCEEDED", profile: "supervised" as const, task: "Repair",
    repo_root: "C:\\repo", current_stage: "final", created_at: "", updated_at: "",
  }];
  render(<Dashboard runs={runs} state="populated" onOpen={vi.fn()} onRetry={vi.fn()} />);
  expect(screen.getAllByTestId("kpi-card")).toHaveLength(4);
  expect(screen.getByRole("img", { name: /run state distribution/i })).toBeVisible();
  expect(screen.getByRole("region", { name: "Live activity" })).toBeVisible();
  expect(screen.getByRole("button", { name: "run-1" })).toBeVisible();
});

it("keeps dashboard distribution buckets mutually exclusive", () => {
  const base: Omit<Run, "id" | "state"> = {
    profile: "supervised" as const, task: "Repair", repo_root: "C:\\repo",
    current_stage: "", created_at: "", updated_at: "",
  };
  render(<Dashboard runs={[
    { ...base, id: "active", state: "DECIDING" },
    { ...base, id: "approval", state: "AWAITING_APPROVAL" },
    { ...base, id: "done", state: "SUCCEEDED" },
  ]} state="populated" onOpen={vi.fn()} onRetry={vi.fn()} />);
  expect(screen.getByText("1 active, 1 awaiting approval, 1 terminal.")).toBeVisible();
});
