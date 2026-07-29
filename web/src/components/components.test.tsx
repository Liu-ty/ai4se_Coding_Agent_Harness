import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "vitest";
import { ApprovalPanel } from "./ApprovalPanel";
import { DiffViewer } from "./DiffViewer";
import { Timeline } from "./Timeline";

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
    digest: "sha256:abc", action: "apply_patch", files: ["src/sum.ts"],
    risk: "Modifies one tracked file",
  }} onDecision={onDecision} />);
  await userEvent.click(screen.getByRole("button", { name: "Approve once" }));
  expect(onDecision).toHaveBeenCalledWith("approve", "sha256:abc");
  expect(screen.queryByText(/always allow|remember/i)).toBeNull();
  expect(screen.getByRole("button", { name: "Approve once" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Reject" })).toBeDisabled();
});

it("renders a read-only keyboard-focusable diff", () => {
  render(<DiffViewer content={"- return 1\n+ return 2"} truncated={false} />);
  expect(screen.getByRole("region", { name: "Read-only diff" })).toHaveAttribute("tabindex", "0");
  expect(screen.queryByRole("textbox")).toBeNull();
});
